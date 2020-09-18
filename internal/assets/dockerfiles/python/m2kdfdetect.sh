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
SPECIAL_FILES=($BASE_DIR/requirements.txt $BASE_DIR/setup.py $BASE_DIR/environment.yml $BASE_DIR/Pipfile)

for fileName in "${SPECIAL_FILES[@]}"
do
   if [ -f "$fileName" ]; then
      startScript=`grep -lRe "__main__" $1 | awk '{print $1}' | xargs -n1 basename`
      echo '{"MAINSCRIPT": "'$startScript'", "APPNAME": "app", "Port": 8080}'
      exit 0
   fi
done

exit 1