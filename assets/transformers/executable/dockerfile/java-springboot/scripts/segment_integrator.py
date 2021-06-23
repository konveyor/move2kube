import ipdb 
import json
from pprint import pprint 
import os

from mappings import (java_versions_mapping, 
                    java_jboss_image_compatibility_mapping, 
                    java_liberty_image_compatibility_mapping,
                    jboss_deployment_paths,
                    java_images_mapping,
                    process_consolidated_images
                    )

class SegmentIntegrator():

    def __init__(self,
        framework_params,
        server_params,
        metadata, 
        pcf_params,
        build_type="maven"
        ):

        self.framework_params = framework_params
        self.server_params    = server_params
        self.metadata         = metadata
        self.pcf_params       = pcf_params
        self.build_type       = build_type
        
        self.segments = {}

    def construct_segment_list(self):

        java2liberty, java2wildfly, liberty2java, wildfly2java = process_consolidated_images()

        self.segments["metadata"] = self.metadata
        self.segments["type"] =  "segments"
        self.segments["segments"] = []

        # Add license segment 
        # -------------------
        self.segments["segments"].append({  
                "segment_id": "segments/dockerfile_license/Dockerfile"
        })
    
        # Identify build type
        # -------------------
        if self.build_type == "maven":


            # Identify framework
            # ------------------
            framework , framework_params = zip(*list(self.framework_params.items()))
            framework = framework[0]
            framework_params = framework_params[0] 

            # Add Maven Build segment
            # ------------------------
            
            java_version = self.metadata["artifacts_versions"]["java.version"]

            self.segments["segments"].append({  
                "segment_id": "segments/dockerfile_maven_build/Dockerfile",
                "app_name": framework_params["app_name"],
                "java_version": java_versions_mapping[java_version]
            })

            if framework == "springboot":
                # -----------
                # Spring boot
                # -----------

                # Check is server is provided or embedded
                # ---------------------------------------
                if self.server_params["is_embedded"] == False:
                    # Server not embedded, we use a custom option

                    if self.server_params["server_type"] == "jboss":  # wildfly

                        candidate_images = java2wildfly[java_version]
                        if len(candidate_images) == 0:
                            jboss_candidate = "JBOSS_DEFAULT"
                        else:
                            jboss_candidate = sorted(candidate_images)[0]

                        # need criteria for this. ideally, we can present the short
                        # list to the user to select

                        # Add Jboss/wildfly Runtime segment
                        # ------------------------
                        self.segments["segments"].append({  
                            "segment_id": "segments/dockerfile_jboss_runtime/Dockerfile",
                            "port": framework_params["port"],
                            "app_file": framework_params["app_file"], 
                            "jboss_image": "wildfly:"+jboss_candidate,
                            "deployment_file": "jhello.war",
                            "deployment_path": jboss_deployment_paths[0]
                        })
                    elif self.server_params["server_type"] == "liberty":
                        # Add an Openliberty Runtime segment
                        # ------------------------

                        candidate_images = java2liberty[java_version]
                        if len(candidate_images) == 0:
                            liberty_candidate = "LIBERTY_DEFAULT"
                        else: 
                            liberty_candidate = sorted(candidate_images)[0]

                        
                        
                        port = ""
                        if "httpEndpoint" in self.server_params and self.server_params["httpEndpoint"] is not None:
                            port = [self.server_params["httpEndpoint"]]
                        else:
                            if "port" in self.framework_params:
                                port = self.framework_params["port"]
                            else:
                                port = 8080
                        
                        self.segments["segments"].append({
                            "segment_id": "segments/dockerfile_openliberty_runtime/Dockerfile",
                            "port": port,
                            "openliberty_image": java_liberty_image_compatibility_mapping[java_version],
                        })

                else:
                    # Server embedded, use default embedded server
                    self.segments["segments"].append({  
                        "segment_id": "segments/dockerfile_springboot_embedded_server_runtime/Dockerfile",
                        "port": framework_params["port"],
                        "app_file": framework_params["app_file"],
                        "java_image" : java_images_mapping[java_version]
                    })
            else:

                if self.server_params["server_type"] == "liberty": 

                    candidate_images = java2liberty[java_version]
                    if len(candidate_images) == 0:
                        liberty_candidate = "LIBERTY_DEFAULT"
                    else: 
                        liberty_candidate = sorted(candidate_images)[0]

                    self.segments["segments"].append({  
                        "segment_id": "segments/dockerfile_openliberty_runtime/Dockerfile",
                        "port": framework_params["port"],
                        "openliberty_image": java_liberty_image_compatibility_mapping[java_version]
                    })

    def persist_full_template(self):

        full_template = []
        for i in self.segments["segments"]: 
            #print(i["segment_id"]) 
            with open(i["segment_id"], "r") as f:
                lines = f.readlines()
                full_template.extend(lines)
                full_template.append("\n")

        # save
        output_path = "./"  # TODO: parameterize?  
        with open(os.path.join(output_path, "Dockerfile-full_template"), "w") as f:
            for l in full_template:
                f.write(l)
        
    def get_json_output(self):
        return json.dumps(self.segments)