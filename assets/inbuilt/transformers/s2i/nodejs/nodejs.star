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

def directory_detect(dir):
    if fs.exists(fs.pathjoin(dir,"package.json")):
        return  {
            "unnamedServices": [{
                    "generates": ["ContainerBuild"],
                    "generatedBases": ["ContainerBuild"],
                    "paths": {
                        "ProjectPath": [dir]
                    },
                    "configs": {
                        "S2IMetadata": {
                            "S2IBuilder":"registry.access.redhat.com/ubi8/nodejs-10"
                        }
                    }
                }]
        }
    return None

def transform(new_artifacts, old_artifacts):
    generated_artifacts = []
    path_mappings = []
    for a in new_artifacts:
        na = {
            "name": a["name"],
            "artifact": "S2IMetadata",
            "paths": a["paths"],
            "configs": a["configs"]
        }
        spm = {
            "type": "Source",
            "destinationPath": "source"
        }
        pm = {
            "type": "Template",
            "destinationPath": "source"
        }
        generated_artifacts.append(na)
        path_mappings.append(spm)
        path_mappings.append(pm)
    return {
        "pathMappings": path_mappings,
        "artifacts": generated_artifacts
    }