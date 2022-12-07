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
import requests
from preprocess import process_cf_services
from pathlib import Path

catalogue = {}


def init_catalogue(cf_services_path="cfservices.yaml", catalogue_path="catalogue.yaml"):
    global catalogue
    if len(catalogue) != 0:
        print("the catalogue has already been initialized")
        return
    process_cf_services(cf_services_path, catalogue_path)
    with open(catalogue_path) as catalogue_file:
        catalogue = yaml.safe_load(catalogue_file)
    print("catalogue ------>", catalogue)


def find_cf_services(newArtifacts):
    global catalogue
    print("trying to find a cf services file among the collected artifacts")
    for artifact in newArtifacts:
        print("checking the artifact", artifact, "to see if it is CollectOutput")
        if artifact["type"] != "CollectOutput":
            continue
        if "paths" not in artifact:
            continue
        if "CollectOutput" not in artifact["paths"]:
            continue
        for collectOutputPath in artifact["paths"]["CollectOutput"]:
            print(
                "parsing the file",
                collectOutputPath,
                "as a Move2Kube collect output YAML file",
            )
            with open(collectOutputPath) as f:
                collectOutput = yaml.safe_load(f)
            if collectOutput["kind"] != "CfServices":
                print(f"not a CfServices file. Actual kind is: {collectOutput['kind']}")
                continue
            print(
                f'initializing the catalogue using the CfServices file at path "{collectOutputPath}"'
            )
            init_catalogue(cf_services_path=collectOutputPath)


# Performs the detection of pom file and extracts service name
def transform(newArtifacts):
    global catalogue
    pathMappings = []
    artifacts = []

    # print("artifactsData:", artifactsData)
    print("Number of new artifacts: " + str(len(newArtifacts)))

    find_cf_services(newArtifacts)
    init_catalogue()

    for artifact in newArtifacts:
        print("checking the artifact", artifact, "to see if it is CollectOutput")
        if artifact["type"] != "CollectOutput":
            continue
        if "paths" not in artifact:
            continue
        if "CollectOutput" not in artifact["paths"]:
            continue
        for collectOutputPath in artifact["paths"]["CollectOutput"]:
            print(
                "parsing the file",
                collectOutputPath,
                "as a Move2Kube collect output YAML file",
            )
            with open(collectOutputPath) as f:
                collectOutput = yaml.safe_load(f)
            if collectOutput["kind"] != "CfApps":
                print(f"not a CfApps file. Actual kind is: {collectOutput['kind']}")
                continue
            print(f'found a CfApps file at path "{collectOutputPath}"')
            if "spec" not in collectOutput:
                continue
            if "applications" not in collectOutput["spec"]:
                continue
            for cloudFoundryApp in collectOutput["spec"]["applications"]:
                try:
                    vcapServiceJSON = cloudFoundryApp["environment"]["systemenv"][
                        "VCAP_SERVICES"
                    ]
                    vcapServices = json.loads(vcapServiceJSON)
                except:
                    print(
                        'failed to find and parse "environment.systemenv.VCAP_SERVICES" in this app, skipping'
                    )
                    continue
                for vcapServiceName in vcapServices:
                    if vcapServiceName not in catalogue:
                        print(
                            f'failed to find the cf vcap service "{vcapServiceName}" in the catalogue'
                        )
                        continue
                    vcapService = catalogue[vcapServiceName]
                    vcapServiceStr = yaml.dump(vcapService)
                    cfServiceName, cfServiceStr = vcapServiceName, vcapServiceStr
                    print(
                        "cfServiceName:",
                        cfServiceName,
                        "cfServiceStr:",
                        cfServiceStr,
                    )
                    try:
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
                        t3 = t2["containerization"][0]["Ref Dockers"]
                        print(t3)
                        print("-" * 50)
                    except Exception as e:
                        print(
                            "failed to standardize and containerize. Error:",
                            e,
                        )
                        continue
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
                    print("new artifact:", new_artifact)
                    artifacts.append(new_artifact)
    return {"pathMappings": pathMappings, "artifacts": artifacts}


# Entry-point of transform script
def main():
    input_path = os.environ["M2K_TRANSFORM_INPUT_PATH"]
    with open(input_path) as f:
        artifactsData = json.load(f)
    output = transform(artifactsData["newArtifacts"])
    output_path = os.environ["M2K_TRANSFORM_OUTPUT_PATH"]
    output_dir = Path(output_path).parent.absolute()
    os.makedirs(output_dir, mode=0o777, exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(output, f)


if __name__ == "__main__":
    main()
