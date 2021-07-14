#!/usr/bin/env bash

#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# Takes as input the source folder and returns error if it is not fit
BASE_DIR=$1
IMAGE="registry.access.redhat.com/ubi8/go-toolset:latest"

if [ ! -f "$1/go.mod" ]; then
   found="$(find "$BASE_DIR"/. -name "*.go" -print -quit | wc -l)"

   if [ "$found" -eq 1 ]; then
      echo '{"generates":"ContainerBuild","generatedBases":"ContainerBuild","builder": "'$IMAGE'", "port": 8080}'
   else
      exit 1
   fi
else
   echo '{"generates":"ContainerBuild","generatedBases":"ContainerBuild","builder": "'$IMAGE'", "port": 8080}'
fi
