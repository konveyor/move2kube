import ipdb
import json
from os.path import join
import xmltodict

from mappings import (
    java_jboss_image_compatibility_mapping,
    java_liberty_image_compatibility_mapping,
    jboss_deployment_paths,
    java_images_mapping,
    process_consolidated_images
)

default_values ={
    "tomcat":  {"port": [8080], "path": "" },
    "jboss":   {"port": [8080], "path": "" }, 
}

class ServerParameterExtractor():

    def __init__(self, captured_data, server_detection_result, metadata):
        self.captured_data = captured_data
        self.server_detection_result = server_detection_result
        self.metadata = metadata

    def extract(self):
    
        java2liberty, java2wildfly, liberty2java, wildfly2java = process_consolidated_images()


        result ={}

        if self.server_detection_result["is_embedded"] == True:            
            result["is_embedded"] = True
        
        else: 
            result["is_embedded"] = False

            # What is the java version?
            java_version = "1.8" # default
            if "java.version" in self.metadata["artifacts_versions"]:
                java_version = self.metadata["artifacts_versions"]["java.version"]

            # Need to identify the type of server used. 
            # For now, lets search for Liberty, if not found, 
            # default option is Jboss/Wildfly

            # Search for Liberty
            liberty_server_file = ""
            
            for i in self.captured_data["app_config_files"]:
                if "server.xml" in i:
                    liberty_server_file = i 

            if liberty_server_file != "":
                # We have liberty

                # Get information from the server.xml file
                result["server_type"] = "liberty"
                with open(liberty_server_file, "r") as f:
                    xcontent = f.read()
                    try:
                        obj = xmltodict.parse(xcontent)
                        if "server" in obj.keys():
                            # httpEndpoint
                            if "httpEndpoint" in obj["server"]:
                                result["httpEndpoint"] = obj["server"]["httpEndpoint"]["@httpPort"]
                            ## application
                            if "application" in obj["server"]:
                                result["application"] = obj["server"]["application"]
                    except:
                        print(" problem processing file")

                # Pick a liberty server 
                # Criteria: [works with the currrent java version]
                

                candidate_images = java2liberty[java_version]
                if len(candidate_images) == 0:
                    liberty_candidate = "LIBERTY_DEFAULT"
                else: 
                    liberty_candidate = sorted(candidate_images)[0]
                    result["compatible_liberty"] = liberty_candidate

                    # Lets load that image data object

                    spef = ServerParameterExtractorFromImage("liberty", liberty_candidate)
                    spef_config = spef.image_config
                    
                    if "User" in spef_config: 
                        result["user"] = spef_config["User"]

                    if "ExposedPorts" in spef_config:
                        result["exposed_port"] = spef_config["ExposedPorts"]

            else:
                # Pick a jboss/wildfly server
                # Criteria: [works with the currrent java version]
                result["server_type"] = "jboss"

                # Which jboss/wildfly servers work with that java version
                
                candidate_images = java2wildfly[java_version]
                if len(candidate_images) == 0:
                    jboss_candidate = "JBOSS_DEFAULT"
                else:
                
                    compatible_jboss = sorted(candidate_images)[-1]
                    result["compatible_jboss"] = compatible_jboss

                    # Lets load that image data object
                    spef = ServerParameterExtractorFromImage("jboss", compatible_jboss)
                    spef_config = spef.image_config

                    if "User" in spef_config: 
                        result["user"] = spef_config["User"]

                    if "ExposedPorts" in spef_config:
                        result["exposed_port"] = spef_config["ExposedPorts"]

        return result 
        
class ServerParameterExtractorFromImage():

    def __init__(self, server_name, image_name):

        # Find image data locally
        self.server_name = server_name
        self.image_name =  image_name

        # load consolidated file
        with open("scripts/image_processor/output/consolidated_images2config.json", "r") as f:
            self.images2config = json.load(f)
 
        self.image_config = {}
        if self.server_name in self.image_config:
            if self.image_name in self.image_config[self.server_name]:
                self.image_config = self.images2config[self.server_name][self.image_name]