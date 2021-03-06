/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*global angular, components*/
components.directive('gapidSatellites', function () {
  "use strict";
  return {
    templateUrl: 'components/gapid-satellites.html',
    scope: {},
    controller: function ($scope, $http, $interval) {
      var refresh
      $scope.update = function() {
        $http.get('/satellites/').success(function(data) { 
          $scope.satellites = data;
        });
      }
      $scope.start = function() {
        $scope.update()
        refresh = $interval($scope.update, 5000);
      }
      $scope.stop = function() {
        $interval.cancel(refresh)
      }
      $scope.$on('$destroy', function() {
        $scope.stop();
      });
      $scope.start()
    },
  };
});