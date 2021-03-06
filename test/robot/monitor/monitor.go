// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monitor

import (
	"context"
	"sync"

	"github.com/google/gapid/core/data/search"
	"github.com/google/gapid/core/data/stash"
	"github.com/google/gapid/test/robot/build"
	"github.com/google/gapid/test/robot/job"
	"github.com/google/gapid/test/robot/master"
	"github.com/google/gapid/test/robot/replay"
	"github.com/google/gapid/test/robot/report"
	"github.com/google/gapid/test/robot/subject"
	"github.com/google/gapid/test/robot/trace"
)

// Managers describes the set of managers to monitor for data changes.
type Managers struct {
	Master  master.Master
	Stash   *stash.Client
	Job     job.Manager
	Build   build.Store
	Subject subject.Subjects
	Trace   trace.Manager
	Report  report.Manager
	Replay  replay.Manager
}

// Data is the live store of data from the monitored servers.
// Entries with no live manager will not be updated.
type Data struct {
	mu   sync.Mutex
	cond *sync.Cond

	Gen *Generation

	Devices  Devices
	Workers  Workers
	Subjects Subjects
	Tracks   Tracks
	Packages Packages
	Traces   Traces
	Reports  Reports
	Replays  Replays
}

type DataOwner struct {
	data *Data
}

func NewDataOwner() DataOwner {
	data := &Data{
		Gen: NewGeneration(),
	}
	data.cond = sync.NewCond(&data.mu)
	return DataOwner{data}
}

func (o DataOwner) Read(rf func(d *Data)) {
	o.data.mu.Lock()
	defer o.data.mu.Unlock()
	rf(o.data)
}

func (o DataOwner) Write(wf func(d *Data)) {
	o.data.mu.Lock()
	defer func() {
		o.data.cond.Broadcast()
		o.data.mu.Unlock()
	}()
	wf(o.data)
}

func (data *Data) Wait() {
	data.cond.Wait()
}

// Run is used to run a new monitor.
// It will monitor the data from all the managers that are in the supplied managers, filling in the data structure
// with all the results it receives.
// Each time it receives a batch of updates it will invoke the update function passing in the manager set being
// monitored and the updated set of data.
func Run(ctx context.Context, managers Managers, owner DataOwner, update func(ctx context.Context, managers *Managers, data *Data) error) error {
	// start all the data monitors we have managers for
	if err := monitor(ctx, &managers, owner); err != nil {
		return err
	}

	owner.Read(func(data *Data) {
		for {
			// Update generation
			data.Gen.Update()
			// Run the update
			if update != nil {
				update(ctx, &managers, data)
			}
			// Wait for new data
			data.Wait()
		}
	})
	return nil
}

func monitor(ctx context.Context, managers *Managers, owner DataOwner) error {
	// TODO: care about monitors erroring
	all := &search.Query{Monitor: true}
	if managers.Job != nil {
		go managers.Job.SearchDevices(ctx, all, owner.updateDevice)
		go managers.Job.SearchWorkers(ctx, all, owner.updateWorker)
	}
	if managers.Build != nil {
		go managers.Build.SearchTracks(ctx, all, owner.updateTrack)
		go managers.Build.SearchPackages(ctx, all, owner.updatePackage)
	}
	if managers.Subject != nil {
		go managers.Subject.Search(ctx, all, owner.updateSubject)
	}
	if managers.Trace != nil {
		go managers.Trace.Search(ctx, all, owner.updateTrace)
	}
	if managers.Report != nil {
		go managers.Report.Search(ctx, all, owner.updateReport)
	}
	if managers.Replay != nil {
		go managers.Replay.Search(ctx, all, owner.updateReplay)
	}

	return nil
}
