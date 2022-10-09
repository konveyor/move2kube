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
import requests


def get_svc_name_and_desc(service):
    del service["guid"]
    del service["active"]
    del service["bindable"]
    del service["extra"]
    del service["planupdateable"]
    del service["updatedat"]
    del service["createdat"]
    del service["instancesretrievable"]
    del service["bindingsretrievable"]
    del service["servicebrokerguid"]
    # print(service)
    # svcstr = ""
    # for k, v in service.items():
    # svcstr += f"{k}: {v}, "
    svcstr = json.dumps(service)
    svcname = service["label"] if "label" in service else "string"
    return svcname, svcstr


# Performs the detection of pom file and extracts service name
def transform(artifactsPath):
    pathMappings = []
    artifacts = []
    with open(artifactsPath) as f:
        artifactsData = json.load(f)
        # print("artifactsData:", artifactsData)
        newArtifacts = artifactsData["newArtifacts"]
        print("Number of new artifacts: " + str(len(newArtifacts)))
        for artifact in newArtifacts:
            print("checking artifact", artifact, "to see if it is CollectOutput")
            if artifact["type"] != "CollectOutput":
                continue
            collectOutputPaths = artifact["paths"]["CollectOutput"]
            if len(collectOutputPaths) == 0:
                print("the artifact has no collect output paths")
                continue
            collectOutputPath = collectOutputPaths[0]
            print(
                "parsing the file",
                collectOutputPath,
                "as a Move2Kube collect output YAML file",
            )
            with open(collectOutputPath) as collectOutputFile:
                collectOutput = yaml.safe_load(collectOutputFile)
                cloudFoundryServices = collectOutput["spec"]["services"]
                for cfService in cloudFoundryServices:
                    # print('cfService:', cfService)
                    cfServiceName, cfServiceStr = get_svc_name_and_desc(cfService)
                    print(
                        "cfServiceName:", cfServiceName, "cfServiceStr:", cfServiceStr
                    )
                    print("standardize with TCA")
                    t1 = requests.post(
                        "http://localhost:8000/standardize",
                        json=[
                            {
                                "application_name": cfServiceName,  # 'App1',
                                "application_description": cfServiceStr,  # 'App1: rhel, db2, java, tomcat',
                                "technology_summary": cfServiceStr,  # 'App1: rhel, db2, java, tomcat'
                            }
                        ],
                    ).json()
                    print(t1)
                    print("containerize with TCA")
                    t2 = requests.post(
                        "http://localhost:8000/containerize",
                        json=t1["standardized_apps"],
                    ).json()
                    print(t2)
                    # artifactName = cfService['label'] if 'label' in cfService else '<unnamed-cf-service>'
                    try:
                        t3 = t2["containerization"][0]["Ref Dockers"]
                    except Exception as e:
                        print(
                            "failed to get the containerization from the response. Error:",
                            e,
                        )
                        continue
                    print(t3)
                    print("-" * 50)
                    operators = {}
                    for op in t3:
                        op_name = op["name"]
                        op_url = op["url"]
                        operators[op_name] = {
                            "url": op_url,
                            "operatorName": op_name,
                            "installPlanApproval": "Manual",
                            "catalogSource": "operatorhubio-catalog",
                            "catalogChannel": "stable",
                        }
                    new_artifact = {
                        "name": cfServiceName,
                        "type": "OperatorsToInitialize",
                        "configs": {
                            "OperatorsToInitializeConfig": {
                                "operators": operators,
                            },
                        },
                    }
                    print("new_art:", new_artifact)
                    artifacts.append(new_artifact)
                # services[filePath] = [{"type": "CollectOutput", "paths": {"ServiceDirectories": [rootDir]} }]
    return {"pathMappings": pathMappings, "artifacts": artifacts}


# Entry-point of transform script
def main():
    services = transform(sys.argv[1])
    outDir = "/var/tmp/m2k_transform_output"
    os.makedirs(outDir, exist_ok=True)
    with open(os.path.join(outDir, "m2k_transform_output.json"), "w+") as f:
        json.dump(services, f)


if __name__ == "__main__":
    main()
