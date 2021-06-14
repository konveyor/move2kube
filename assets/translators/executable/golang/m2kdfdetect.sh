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
<<<<<<< HEAD:internal/assets/dockerfiles/golang/m2kdfdetect.sh
#   limitations under the License.

# Takes as input the source directory and returns error if it is not fit
found=$(find "$1"/. -name "*.go" -print -quit | wc -l)

if [ ! "$found" -eq 1 ]; then
    exit 1
fi

echo '{"port": 8080, "app_name": "app-bin"}'
=======
#   limitations under the License.
>>>>>>> feat: Allow containerizers to run as container or locally depending on environment:internal/assets/dockerfiles/java/segments/dockerfile_license/Dockerfile
