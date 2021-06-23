import ipdb
import re
import yaml

from maven_processor import PomProcessor
from default_params import default_params

default_values ={
    "springboot": {"port": [8080], "path": "" },
    "liberty": {"port": [8090], "port_https": 9443, "path": "" }
}

class FrameworkParameterExtractor():

    def __init__(self, captured_data, basename):

        self.captured_data = captured_data
        self.basename = basename

    def extract(self, framework):

        if framework == "springboot":
            return self.extract_springboot()

        elif framework  == "liberty":
            return self.extract_liberty()
    
    def extract_springboot(self):
        """
        Spring boot should return:
        - port
        - app_name (the app path)
        - app_file (the name of the built jar/war file)
        """
        output = {
            "port": None,
            "app_file": None,
            "app_name": None
        }

        # ---- 
        # port
        # ----
        
        # inspect *.yml 
        ymls =  self.captured_data["application_yml_files"]
        
        ports_from_yml = []
        if len(ymls)> 0:
            for yml in ymls:
                with open(yml, "r") as f:
                    yml_data = yaml.load(f)
                    if "server" in yml_data:
                        if "port" in yml_data["server"]:
                            ports_from_yml.append(yml_data["server"]["port"])
    
        # inspect *.properties files  
        props =  self.captured_data["application_properties_files"]
        ports_from_prop = []
        if len(props)> 0:
            for prop in props:
                with open(prop, "r") as f:
                    candidate_lines  = [ l for l in f.readlines() if "=" in l]
                    lines = [l.strip().split("=") for l in candidate_lines]
                    #ipdb.set_trace()
                    for k,v in lines:
                        if "port" in k.split("."):
                            ports_from_prop.append(v)

        # combine them 
        output["port"] =  list(set(ports_from_yml +  ports_from_prop))

        # use default values if none was found
        if len(output["port"]) == 0:
            output["port"].append(default_params["springboot"]["port"])
    
        # ----
        # app_name
        # ----
        output["app_name"] = self.basename

        # ----
        # app file
        # ----
        pom_path = self.captured_data["maven_build_automation_files"][0]
        pp = PomProcessor(pom_path)
        app_file = pp.get_generated_app_file()
        output["app_file"] = app_file

        return output

    def extract_liberty(self):
        """
        Liberty should return :
        - port
        - app name 
        """
        return None

class FrameworkParameterExtractorHeuristic():

    def __init__(self, captured_data):
        self.captured_data = captured_data

    def extract(self):
        """

        """

        output = {
            "port": None,
            "app_file": None,
            "app_name": None
        }

        # merge files stored in captured_data into a single list
        all_files = []
        for k, v in self.captured_data.items():
            if k in ["app_config_files", "application_properties_files",
        "application_yml_files","maven_build_automation_files", 
        "gradle_build_automation_files"]:
                all_files.extend(v)
        all_files =  list(set(all_files))

        # inspect every file looking for ports (ex):
        port_candidates = {} 
        path_candidates = {}

        for fi in all_files:
            with open(fi, "r") as f:
                fi_lines = [(i,l) for i,l in enumerate(f.readlines())] 

            port_candidates_lines = [(i,l) for i,l in fi_lines if "port" in l]
            if len(port_candidates_lines) > 0:
                port_candidates[fi] = port_candidates_lines

            path_candidates[fi] = fi_lines

        # check port candidate files
        feasible_ports = []
        for k,v in port_candidates.items():
            for line in v:
                clean_v =  line[1].strip()
                feasible_port =re.findall("\d{4}", clean_v) 
                if len(feasible_port) > 0:
                    feasible_ports.extend(feasible_port)
                
        feasible_ports = list(set(feasible_ports))
        feasible_ports = list(map(int, feasible_ports))

        output["port"] = feasible_ports

        return output