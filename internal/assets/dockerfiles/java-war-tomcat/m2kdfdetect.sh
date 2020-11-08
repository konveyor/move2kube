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
main() {
    [ "$#" -gt 1 ] && echo 'multiple WAR files. exiting' && exit 1
    [ ! -e "$1" ] && echo 'no WAR files. exiting' && exit 1
    printf '{"port":8080, "war_path":"%s"}' "$(basename "$1")"
}

main "$1/"*.war
