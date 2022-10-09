#!/usr/bin/env python

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

import sys
import os
import json
import yaml

# Performs the detection of pom file and extracts service name
def detect(inputPath):
    print('detect start inputPath:', inputPath)
    services = {}
    with open(inputPath) as f:
        detectInput = json.loads(f.read())
        print('detectInput', detectInput)
        for rootDir, _, fileList in os.walk(detectInput["InputDirectory"]):
            for filePath in fileList:
                print(
                    "checking to see if the file",
                    filePath,
                    "is a Move2Kube collect output",
                )
                if not filePath.endswith(".yaml"):
                    continue
                fullFilePath = os.path.join(rootDir, filePath)
                with open(fullFilePath) as fff:
                    collectFile = yaml.safe_load(fff)
                    # print("collectFile:", collectFile)
                    if (
                        "apiVersion" not in collectFile
                        or collectFile["apiVersion"] != "move2kube.konveyor.io/v1alpha1"
                    ):
                        print(
                            "apiVersion does not match. Expected: 'move2kube.konveyor.io/v1alpha1' Actual: {}".format(
                                collectFile["apiVersion"]
                            )
                        )
                        continue
                    if "kind" not in collectFile or collectFile["kind"] != "CfServices":
                        print(
                            "kind does not match. Expected: 'CfServices' Actual: {}".format(
                                collectFile["kind"]
                            )
                        )
                        continue
                    print("found a Move2Kube collect output YAML file")
                    # print(collectFile)
                    artifactName = os.path.basename(fullFilePath)
                    services[artifactName] = [
                        {
                            "name": artifactName,
                            "type": "CollectOutput",
                            "paths": {"CollectOutput": [fullFilePath]},
                        }
                    ]
    # print('detect end services:', services)
    return services


# Entry-point of detect script
def main():
    services = detect(sys.argv[1])
    outDir = "/var/tmp/m2k_detect_output"
    os.makedirs(outDir, exist_ok=True)
    with open(os.path.join(outDir, "m2k_detect_output.json"), "w+") as f:
        json.dump(services, f)


if __name__ == "__main__":
    main()
