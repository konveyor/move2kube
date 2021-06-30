import json
import ipdb
import yaml

try:
    from yaml import CLoader as Loader, CDumper as Dumper
except ImportError:
    from yaml import Loader, Dumper

class PCFExtractor():

    def __init__(self, captured_data):
        self.captured_data = captured_data

    def extract(self):

        result ={
            "cmd" : {},
            "memory": {}
        }

        if "yml_files" in self.captured_data:
            ymls =  self.captured_data["yml_files"]

            if len(ymls)> 0:
                for yml in ymls:
                    if "manifest" in yml:
                        with open(yml, "r") as f:
                            yml_data_stream = yaml.load_all(f, Loader=Loader)

                            result["memory"][yml] = []
                            result["cmd"][yml] = []
                            for  yml_data in yml_data_stream:
                                res_memory, res_cmd = [], []

                                if yml_data is None:
                                    continue
                                if "applications" in yml_data:
                                    for app in yml_data["applications"]:
                                        name = app["name"]

                                        if "memory" in app:
                                            memory = app["memory"]
                                        else: 
                                            memory = ""

                                        if "command" in app:
                                            cmd = app["command"]
                                        else: 
                                            cmd = ""

                                        res_memory.append({
                                            "name": name, "memory": memory
                                        })


                                        res_cmd.append({
                                            "name": name, "cmd": cmd
                                        })



                            result["memory"][yml].append(res_memory) 
                            result["cmd"][yml].append(res_cmd) 

                    else:
                        # could be anything else
                        # TODO dedice what to do
                        continue
                    
        return result