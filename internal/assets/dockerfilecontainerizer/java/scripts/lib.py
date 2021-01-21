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

import os
import argparse
import ipdb
import json
import xmltodict

from collections import defaultdict
from os import listdir
from os.path import isfile, join
from pprint import pprint


default_values ={
    "tomcat":  {"port": 8080, "path": "" },
    "jboss":   {"port": 8080, "path": "" }, 
    "liberty": {"port": 8090, "port_https": 9443, "path": "" }
}

def load_captured_data(output_path):

    data = defaultdict(list)
    data["build_type"] = "undefined"

    if isfile(join(output_path, "app_config_files.output")):
        with open(join(output_path, "app_config_files.output"), "r") as f:
            data_app_config_files = [l.strip() for l in f.readlines()]
        if len(data_app_config_files) > 0:
            data["app_config_files"] = data_app_config_files

    if isfile(join(output_path, "gradle_build_automation_files.output")):
        with open(join(output_path, "gradle_build_automation_files.output"), "r") as f:
            data_gradle_build_automation_files = [l.strip() for l in f.readlines()]     
        if len(data_gradle_build_automation_files) > 0:
            data["gradle_build_automation_files"] = data_gradle_build_automation_files
            data["build_type"] = "gradle"


    if isfile(join(output_path, "maven_build_automation_files.output")):
        with open(join(output_path, "maven_build_automation_files.output"), "r") as f:
            data_maven_build_automation_files = [l.strip() for l in f.readlines()]     
        if len(data_maven_build_automation_files) > 0:
            data["maven_build_automation_files"] = data_maven_build_automation_files
            data["build_type"] = "maven"


    if isfile(join(output_path, "gradle_dependencies.output")):
        with open(join(output_path, "gradle_dependencies.output"), "r") as f:
            data["gradle_dependencies"] = [l.strip() for l in f.readlines()]
    
    return data


def get_server_app(captured_data, app_path, output_path):

    # initial values
    output =  {
        "server_app": "undefined",
        "version": "undefined"
    }

    version =""

    server_voting = {"tomcat": 0, "liberty": 0, "jboss": 0}
    folder_voting = {"tomcat": 0, "liberty": 0, "jboss": 0}

    # folder structure
    src_path =  join(app_path, "src")
    folders = [x[0] for x in os.walk(src_path)]
    for f in folders:
        for k in folder_voting.keys():
            if k in f:
                folder_voting[k]+=1

    # build automation 
    if "maven_build_automation_files" in captured_data:
        for xf in captured_data["maven_build_automation_files"]:
            if "pom.xml" in xf:        
                with open(xf, "r") as f:
                    xcontent = f.read()
                    obj = xmltodict.parse(xcontent)
                    
                    # dependencies
                    if "dependencies" in obj["project"]:
                        for i in  obj["project"]["dependencies"]["dependency"]: 
                            for k in server_voting.keys():
                                if k in json.dumps(i):
                                    server_voting[k]+=1
                    # build
                    if "build" in obj["project"]:
                        if "plugins" in obj["project"]["build"]:
                            for i in obj["project"]["build"]["plugins"]["plugin"]:
                                for k in server_voting.keys():
                                    if k in json.dumps(i):
                                        server_voting[k]+=1
                    # properties
                    if "properties" in obj["project"]:
                        for i in obj["project"]["properties"]:
                            for k in server_voting.keys():
                                if k in json.dumps(i):
                                    server_voting[k]+=1

    if "gradle_dependencies" in captured_data:
        content = captured_data["gradle_dependencies"]

        # tomcat
        tomcat_content = [l for l in content if "tomcat" in l]
        # jboss
        jboss_content = [l for l in content if "jboss" in l]
        # liberty
        liberty_content = [l for l in content if "liberty" in l]

        if len(tomcat_content) >0:
            server_voting["tomcat"]+=1
        if len(jboss_content) > 0:
            server_voting["jboss"]+=1
        if len(liberty_content) > 0: 
            server_voting["liberty"]+=1

        # liberty - version
        for i in liberty_content:
            if "\---" in i:
                version = i.replace("\---", "")

    # deciding winner and returning data
    global_voting = {}
    for k in server_voting.keys():
        global_voting[k]= server_voting[k] + folder_voting[k] 

    global_voting_reversed = defaultdict(list)
    for k,v in global_voting.items():
        global_voting_reversed[v].append(k)

    server_app, score = list(sorted(global_voting.items(), 
    key= lambda x:x[1], reverse=True))[0]
    
    if len(global_voting_reversed[score]) > 1:
        server_app  = "undefined"

    output["server_app"] = server_app
    output["version"] = version
    return output

def get_app_attributes(captured_data, app_path, output_path, basename, app_name):

    if app_name != "undefined":
        app_data = default_values[app_name]
        final_output = {"port": app_data["port"], 
                        "app_name": app_data["path"]}
    else:
        final_output = {"port": 8080, "app_name": "./"}

    #final_output = {"port": 8080, "app_name": "./"}

    output ={}
    app_config_files = captured_data["app_config_files"]

    path = None

    #print("processing candidate server.xml files")
    cfs = [i for i in app_config_files if "server.xml" in i]
    if len(cfs) > 0:
        config_result = {}
        for cf in cfs:
            cf_result = {}
            with open(cf , "r") as f:
                    xcontent = f.read()
                    try:
                        obj = xmltodict.parse(xcontent)
                        if "server" in obj.keys():
                            # httpEndpoint
                            if "httpEndpoint" in obj["server"]:
                                cf_result["httpEndpoint"] = obj["server"]["httpEndpoint"]
                            # application
                            if "application" in obj["server"]:
                                cf_result["application"] = obj["server"]["application"]
                    except:
                        print(" problem processing file")
            config_result[cf] = cf_result
        output["server"] = config_result
    
    #print("processing candidate context.xml files")
    cfs = [i for i in app_config_files if "context.xml" in i]
    if len(cfs) > 0:
        context_result = {}
        for cf in cfs:
            #cf_result = {}
            with open(cf , "r") as f:
                    xcontent = f.read()
                    obj = xmltodict.parse(xcontent) 
            context_result[cf] = obj
        output["context"] = context_result

    #print("processing candidate root.xml files")
    ros = [i for i in app_config_files if "root.xml" in i]
    if len(ros)> 0:
        root_result = {}
        for ro in ros:
            with open(ro , "r") as f:
                xcontent = f.read()
                obj = xmltodict.parse(xcontent)
            root_result[ro] = obj
        output["root"] = root_result

    # deciding what to include
    if "server" in output and  len(output["server"])  == 1:
        # no need to resolve, just extract values
        server_data = list(output["server"].values())[0]
        if "httpEndpoint" in server_data:
            if "@httpsPort" in server_data["httpEndpoint"]:
                if server_data["httpEndpoint"]["@httpsPort"].isnumeric():
                    final_output["port"] = server_data["httpEndpoint"]["@httpsPort"]

        #if "application" in server_data:
        #    if "@location" in server_data["application"]:
        #        to_inject["app_name"]

    if path is None:
        final_output["app_name"] = basename+"/"

    return final_output


def get_segments( input_type, build_type, server_app, output):
    
    segments = dict()
    segments["type"] =  "segments"
    segments["segments"] = [] 
    sc = 0

    # add license segment 
    segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_license/Dockerfile",
                "files_to_copy" : ["test_file.txt", "test_folder" ]
            })
    sc+=1

    if build_type == "maven":

        segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_maven_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                #"files_to_copy" : []
            })
        sc+=1
        
    elif build_type  == "gradle":

        segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_gradle_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                #"files_to_copy" : []
            })
        sc+=1

    if server_app  == "undefined" or server_app == "liberty":

        segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_liberty_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"],
                #"files_to_copy" : []
            })
        sc+=1

    return segments


def main(args):

    captured_data = load_captured_data(args.output_path)
    output = {}

    # task 1 obtain server app
    output["server_app"] = get_server_app(
        captured_data, 
        args.app_path,
        args.output_path
        )

    # task 2 obtain app attributes
    output["app_attributes"] = get_app_attributes(
        captured_data, 
        args.app_path,
        args.output_path,
        args.basename,
        output["server_app"]["server_app"]
        )

    output["captured_data"] = {}
    for k,v in captured_data.items():
        output["captured_data"][k] = v

    output["input_type"] = args.input_type
    
    input_type = output["input_type"] # file, folder
    build_type = output["captured_data"]["build_type"] # maven, gradle
    server_app = output["server_app"]["server_app"] # liberty, tomcat , jboss,...

    # print the final result as json string
    segments = get_segments(input_type, build_type, server_app, output)
    print(json.dumps(segments))
    
if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--app_path", type=str, default="")
    parser.add_argument("--output_path", type=str, default="")
    parser.add_argument("--basename", type=str, default="")
    parser.add_argument("--input_type", type=str, default="")
    args = parser.parse_args()
    main(args)