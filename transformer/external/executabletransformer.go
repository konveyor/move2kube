/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package external

var (
	// DetectContainerOutputDir is the directory where external transformer detect output is stored
	DetectContainerOutputDir = "/var/tmp/m2k_detect_output"
	// TransformContainerOutputDir is the directory where external transformer transform output is stored
	TransformContainerOutputDir = "/var/tmp/m2k_transform_output"
)
