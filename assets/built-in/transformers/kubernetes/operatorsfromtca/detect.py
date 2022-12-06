#!/usr/bin/env python

#   Copyright IBM Corporation 2022
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

import os
import json
import yaml
from pathlib import Path

# Performs the detection of pom file and extracts service name
def detect(input_dir):
    services = {}
    print("input_dir", input_dir)
    for rootDir, _, fileList in os.walk(input_dir):
        for filePath in fileList:
            print(
                f"checking to see if the file {filePath} is a Move2Kube collect output"
            )
            if not filePath.endswith(".yaml"):
                continue
            fullFilePath = os.path.join(rootDir, filePath)
            with open(fullFilePath) as f:
                collectFile = yaml.safe_load(f)
            # print("collectFile:", collectFile)
            if (
                "apiVersion" not in collectFile
                or collectFile["apiVersion"] != "move2kube.konveyor.io/v1alpha1"
            ):
                print(
                    f"apiVersion does not match. Expected: 'move2kube.konveyor.io/v1alpha1' Actual: {collectFile['apiVersion']}"
                )
                continue
            if "kind" not in collectFile or (
                collectFile["kind"] != "CfApps" and collectFile["kind"] != "CfServices"
            ):
                print(
                    f"kind does not match. Expected: 'CfApps' or 'CfServices' Actual: {collectFile['kind']}"
                )
                continue
            print("found a CfApps or CfServices collect output YAML file")
            # print(collectFile)
            artifactName = os.path.basename(fullFilePath)
            services[artifactName] = [
                {
                    "name": artifactName,
                    "type": "CollectOutput",
                    "paths": {"CollectOutput": [fullFilePath]},
                }
            ]
    return services


# Entry-point of detect script
def main():
    input_path = os.environ["M2K_DETECT_INPUT_PATH"]
    with open(input_path) as f:
        detectInput = json.load(f)
    output = detect(detectInput["InputDirectory"])
    output_path = os.environ["M2K_DETECT_OUTPUT_PATH"]
    output_dir = Path(output_path).parent.absolute()
    os.makedirs(output_dir, mode=0o777, exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(output, f)


if __name__ == "__main__":
    main()
