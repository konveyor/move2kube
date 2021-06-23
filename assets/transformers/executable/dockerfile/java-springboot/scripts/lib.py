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

import re
import os
import argparse
import ipdb
import json
import xmltodict
import yaml

from collections import defaultdict
from os import listdir
from os.path import isfile, join
from pprint import pprint


default_values ={
    "tomcat":  {"port": [8080], "path": "" },
    "jboss":   {"port": [8080], "path": "" }, 
    "liberty": {"port": [8090], "port_https": 9443, "path": "" }
}


# This function standarizes the list of files captured by the bash 
# script. 
# There are 5 categories:
# - app_config_files : any xml file found
# - application_properties_files: any application.properties found
# - application_yml_files: any application.yml or application.yaml found
# - gradle_build_automation_files : any *.gradle found
# - maven_build_automation_files: any pom.xml found (could have overlap with app_config_files)
# - gradle_dependencies: not a file, but the dumped output of gradle  
def load_captured_data(output_path):

    data = defaultdict(list)
    data["build_type"] = "undefined"


    if isfile(join(output_path, "app_config_files.output")):
        with open(join(output_path, "app_config_files.output"), "r") as f:
            data_app_config_files = [l.strip() for l in f.readlines()]
        if len(data_app_config_files) > 0:
            data["app_config_files"] = data_app_config_files


    if isfile(join(output_path, "application_properties_files.output")):
        with open(join(output_path, "application_properties_files.output"), "r") as f:
            data_application_properties_files = [l.strip() for l in f.readlines()]
        if len(data_application_properties_files) > 0:
            data["application_properties_files"] = data_application_properties_files


    if isfile(join(output_path, "application_yml_files.output")):
        with open(join(output_path, "application_yml_files.output"), "r") as f:
            data_application_yml_files = [l.strip() for l in f.readlines()]
        if len(data_application_yml_files) > 0:
            data["application_yml_files"] = data_application_yml_files
    

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



# This function is responsible to detecting the  server/ framework 
# the target application is using 
# It works in a cascaded way. Firstly, it uses a structure-based approach 
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

def get_app_file(
    captured_data, app_path, output_path, basename, app_name 
):

    poms = captured_data["maven_build_automation_files"]

    if len(poms) ==1 :
        pom_file = poms[0]
        with open(pom_file, "r") as f:
            pom_content = f.read()
            obj = xmltodict.parse(pom_content)
            project = obj["project"]
            name = project["name"]
            version = project["version"]
            nv = name + "-" +version
            if "packaging" in project:
                nv = nv + ".war"
            else: 
                nv = nv + ".jar"
        
        return nv
    return False

def get_app_attributes_free_form(
    captured_data, app_path, output_path, basename, app_name 
):

    # merge files into a single list
    all_files = []
    for k, v in captured_data.items():
        if k in ["app_config_files", 
        "application_properties_files",
        "application_yml_files",
        "maven_build_automation_files", 
        "gradle_build_automation_files"]:
    
            all_files.extend(v)
    all_files =  list(set(all_files))

    ## tag analysis
    """
    print("Tag level: ")
    print("===========")

    import xml.etree.ElementTree as ET
    file2tags = {}
    for fi in all_files:
        try:
            xmlTree = ET.parse(fi)
        except:
            print("cannot parse", fi)
            continue
        element_list = []
        for elem in xmlTree.iter():
            element_list.append(elem.tag)

        element_list = list(set(element_list))
        file2tags[fi] = element_list

    for k, v in file2tags.items():
        print(k)
        for vv in v: 
            print(" xml tag: ", vv)
            vv_clean = clean_xml_tag(vv)
            vv_tokens = []
            for i in vv_clean: 
                vv_tokens.extend(i.split("."))
            print("  ", "tokens =>", vv_tokens)

        print("---------")
    """
    #print("Free form: ")
    #print("===========")

    # inspect every file looking for ports (ex):
    port_candidates = {} 
    path_candidates = {}

    for fi in all_files:
        with open(fi, "r") as f:
            fi_lines = [(i,l) for i,l in enumerate(f.readlines())] 

        port_candidates_lines = [(i,l) for i,l in fi_lines if "port" in l]
        if len(port_candidates_lines) > 0:
            port_candidates[fi]   = port_candidates_lines

        path_candidates[fi] = fi_lines

    # check port candidate files

    feasible_ports = []
    for k,v in port_candidates.items():
        #print(k)
        for line in v:
            clean_v =  line[1].strip()
            #print(" ", line)
            feasible_port =re.findall("\d{4}", clean_v) 
            if len(feasible_port) > 0:
                #print(feasible_port)
                feasible_ports.extend(feasible_port)
            #print("     ", re.findall("\d{4}", clean_v)) # can be changed to a range, ex: "\d{4,6}"
        #print("---------")

    feasible_ports = list(set(feasible_ports))
    feasible_ports = list(map(int, feasible_ports))

    """
    # check path candidate files
    for k, v in path_candidates.items():
        print(k)
        for line in v:
            #print(len(line))
            clean_v = line[1].strip()
            print(" ", clean_v)
            print("     ",  re.findall(r'(\/.*?\.[\w:]+)', clean_v))
        print("---------")

    """
    
    return feasible_ports

def clean_xml_tag(tag):
    #tag = tag.lower()
    tag = re.sub(r"\s*{.*}\s*", "", tag)
    return case_split(tag)


def case_split(tag, lower=True):

    l= re.sub('([A-Z][a-z]+)', r' \1', re.sub('([A-Z]+)', r' \1', tag)).split()
    if lower:
        return [t.lower() for t in l]
    else: return l



def get_app_attributes(captured_data, app_path, output_path, basename, app_name):

    if app_name != "undefined":
        app_data = default_values[app_name]
        final_output = {"port": app_data["port"], 
                        "app_name": app_data["path"]}
    else:
        final_output = {"port": [], "app_name": "./"}

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
                    final_output["port"] = [server_data["httpEndpoint"]["@httpsPort"]]

        #if "application" in server_data:
        #    if "@location" in server_data["application"]:
        #        to_inject["app_name"]

    ymls =  captured_data["application_yml_files"]
    ports_from_yml = []
    if len(ymls)> 0:
       
        for yml in ymls:
            with open(yml, "r") as f:
                yml_data = yaml.load(f)
                if "server" in yml_data:
                    if "port" in yml_data["server"]:
                        ports_from_yml.append(yml_data["server"]["port"])
    
    ports_from_yml = list(set(ports_from_yml))
     
    if path is None:

        final_output["app_name"] = basename

    final_output["port"].extend(ports_from_yml)


    return final_output


def get_segments( input_type, build_type, server_app, output, free_form_ports, app_file):
    
    segments = dict()
    segments["type"] =  "segments"
    segments["segments"] = [] 
    sc = 0

    # add license segment 
    segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_license/Dockerfile",
            #    "files_to_copy" : ["test_file.txt", "test_folder" ]
            })
    sc+=1

    if build_type == "maven":

        segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_maven_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "app_file": app_file
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

        final_port = []
        if len(output["app_attributes"]["port"]) == 0:
            final_port.extend(free_form_ports)
        else:
            final_port.extend(output["app_attributes"]["port"])

        
        if len(final_port) == 0:
            final_port.extend("8080")

        segments["segments"].append(
             {  
                "segment_id": "segments/dockerfile_liberty_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": final_port,
                "app_file": app_file
                #"files_to_copy" : []
            })
        sc+=1

    return segments

def get_segments_list_format_flat( input_type, build_type, server_app, output, free_form_ports):
    # gives all posible combination as flat lists

    segments = dict()
    segments["type"] =  "segments"
    segments["segments"] = [] 
    
    build2segment = {
        "maven": {
                "segment_id" : "segments/dockerfile_maven_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "segment_type": "build"
                },
        "gradle": {
                "segment_id": "segments/dockerfile_gradle_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                 "segment_type": "build"
        }
    }

    server2segment = {
        "jboss":{  
                "segment_id": "segments/dockerfile_jboss_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"] + free_form_ports,
                "segment_type": "server"
               
            },
        "liberty":{  
                "segment_id": "segments/dockerfile_liberty_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"] + free_form_ports,
                "segment_type": "server"
               
            },
        "tomcat": {  
                "segment_id": "segments/dockerfile_tomcat_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"] + free_form_ports,
                 "segment_type": "server"
            }
    }

    for build, build_segment in build2segment.items():
        for server , server_segment in server2segment.items():

            segments["segments"].append([  build_segment, server_segment ])

    return segments


def get_segments_list_format_two_levels( input_type, build_type, server_app, output, free_form_ports):
    # gives all posible combination as flat lists

    segments = dict()
    segments["type"] =  "segments"
    segments["segments"] = [] 
    
    build2segment = {
        "maven": {
                "segment_id" : "segments/dockerfile_maven_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "segment_type": "build"
                },
        "gradle": {
                "segment_id": "segments/dockerfile_gradle_build/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                 "segment_type": "build"
        }
    }

    
    server2segment = {
        "jboss":{  
                "segment_id": "segments/dockerfile_jboss_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"] + free_form_ports,
                "segment_type": "server"
               
            },
        "liberty":{  
                "segment_id": "segments/dockerfile_liberty_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"] + free_form_ports,
                "segment_type": "server"
               
            },
        "tomcat": {  
                "segment_id": "segments/dockerfile_tomcat_runtime/Dockerfile",
                "app_name": output["app_attributes"]["app_name"],
                "port": output["app_attributes"]["port"] + free_form_ports,
                 "segment_type": "server"
            }
    }

    for build, build_segment in build2segment.items():

        temp = {
            "build": build_segment,
            "servers":[]
        }

        for server , server_segment in server2segment.items():
            temp["servers"].append(server_segment)

        segments["segments"].append(temp)

    return segments


def check_multimodule(
    captured_data, 
    app_path,
    output_path,
    basename
    ):

    #check maven case:
    modules = []
    module_data = dict()
    if "maven_build_automation_files" in captured_data:

        root_based = app_path + "/pom.xml"

        if root_based in captured_data["maven_build_automation_files"]:
            # we need to open it and check 
            with open(root_based, "r") as f:
                xcontent = f.read()
                obj = xmltodict.parse(xcontent)

            if "project" in obj:
                if "modules" in obj["project"]:
                    modules = obj["project"]["modules"].values()


    modules = list(modules)

    if len(modules) == 0:
        return module_data
    else:
        modules = modules[0]
   
    for m in modules:
        #print("module:", m)

        module_data[m] = {
            "is_executable": False, 
            "depends_on":  []
        }
            
        path_to_pom = join(app_path, m, "pom.xml")
        #print(" ", path_to_pom)

        if isfile(path_to_pom):

            with open(path_to_pom, "r") as f:
                xcontent = f.read()
            obj = xmltodict.parse(xcontent)
            
            # 1. check dependencies from other modules
            if "dependencies" in obj["project"]:
                dependencies = obj["project"]["dependencies"]["dependency"] # list or ordered dict 
                

                if isinstance(dependencies, list):
    
                    for dep in dependencies :
                        id = dep["artifactId"]
                        if id in modules and id != m:
                            #print(id)
                            module_data[m]["depends_on"].append(id)
                else: # its a dict
                    id = dependencies["artifactId"]
                    if id in modules and id != m:
                        #print(id)
                        module_data[m]["depends_on"].append(id)
                     
            # 2. check if the module is executable
            if "build" in obj["project"]:
                # is executable 
                module_data[m]["is_executable"] = True

            #pprint(module_data)
            

    return module_data

def main(args):

    captured_data = load_captured_data(args.output_path)
    output = {}

    # task 0 check if the app is multi-module
    #module_data = check_multimodule(
    #    captured_data, 
    #    args.app_path,
    #    args.output_path,
    #    args.basename)
    #pprint(module_data)

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


     # task 3 obtain app attributes free form 

    free_form_ports  = get_app_attributes_free_form(
        captured_data, 
        args.app_path,
        args.output_path,
        args.basename,
        output["server_app"]["server_app"]
    )


    app_file = get_app_file(
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
    segments = get_segments(input_type, build_type, server_app, output, free_form_ports, app_file)
    #segments = get_segments_list_format_two_levels(input_type, build_type, server_app, output, free_form_ports)

    print(json.dumps(segments))
    
if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--app_path",    type=str, default="Absolute path of the original application")
    parser.add_argument("--output_path", type=str, default="Location of the files generate by the bash script")
    parser.add_argument("--basename",    type=str, default="Basename of the project")
    parser.add_argument("--input_type",  type=str, default="Can be `directory or `file`")

    args = parser.parse_args()
    main(args)