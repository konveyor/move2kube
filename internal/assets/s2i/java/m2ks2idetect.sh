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
NATIVE_IMAGE="registry.access.redhat.com/redhat-openjdk-18/openjdk18-openshift:latest"
WEB_IMAGE="registry.access.redhat.com/jboss-eap-6/eap64-openshift:latest"

# Gradle not supported yet
if [ -f "$1/build.gradle" ]; then
   exit 1
fi

# Ant not supported yet
if [ -f "$1/build.xml" ]; then
   exit 1
fi

if [ -f "$1/pom.xml" ]; then
   echo '{"Builder": "'$WEB_IMAGE'", "Port": 8080}'
   exit 0
fi

found=`find $BASE_DIR/. -name "*.java" -print -quit | wc -l`

if [ $found -eq 1 ]; then
    echo '{"Builder": "'$NATIVE_IMAGE'", "Port": 8080}'
else 
    exit 1
fi