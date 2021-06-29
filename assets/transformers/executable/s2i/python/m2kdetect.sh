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
BASE_DIR="$1"
SPECIAL_FILES=("$BASE_DIR"/requirements.txt "$BASE_DIR"/setup.py "$BASE_DIR"/environment.yml "$BASE_DIR"/Pipfile)
IMAGE="registry.access.redhat.com/rhscl/python-36-rhel7:latest"

for fileName in "${SPECIAL_FILES[@]}"; do
   if [ -f "$fileName" ]; then
      main_script_path="$(grep -lRe "__main__" "$BASE_DIR" | awk '/.py$/ {print}' | head -n 1)"
      main_script_rel_path="$(realpath --relative-to="$BASE_DIR" "$main_script_path")"
      printf '{"builder": "'$IMAGE'", "app_file": "%s", "app_name": "app", "port": 8080}' "$main_script_rel_path"
      exit 0
   fi
done

exit 1
