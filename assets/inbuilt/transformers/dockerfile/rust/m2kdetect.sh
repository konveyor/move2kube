#!/usr/bin/env bash
#   Copyright IBM Corporation 2021
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


# Takes as input the source directory and returns error if it is not fit
BASE_DIR=$1
if [ ! -f "$BASE_DIR/Cargo.toml" ]; then
   exit 1
fi

name=$(awk -F'[ ="]+' '$1 == "name" { print $2 }' $BASE_DIR/Cargo.toml)
echo '{"port": 8080, "app_name": '"\"$name\""'}'
