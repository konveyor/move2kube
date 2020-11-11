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

# Takes as input the source directory and returns error if it is not fit
if [ ! -f "$1/package.json" ]; then
   exit 1
fi

IMAGE="registry.access.redhat.com/ubi8/nodejs-10"
echo '{"builder": "'$IMAGE'", "port": 8080}'
