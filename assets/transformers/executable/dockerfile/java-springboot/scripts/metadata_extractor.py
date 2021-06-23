import ipdb
from maven_processor import PomProcessor
from os.path  import isfile, join

class MetadataExtractor():
    """
    Identifies information associated to the application, but 
    not specific build/deploy parameters
    """

    def __init__(self, captured_data, app_path, basename):
        self.captured_data = captured_data
        self.app_path = app_path 
        self.basename = basename

        if len(self.captured_data["maven_build_automation_files"]) > 0 :
            self.pom_path = self.captured_data["maven_build_automation_files"][0]
            self.pp = PomProcessor(self.pom_path)

    
    def get_all_metadata(self):

        artifacts_versions = self.get_artifacts_versions()
        hierarchy_info     = self.get_hierarchy()
        packaging          = self.get_packaging()
        csa                = self.get_config_server_availability()
        plugins            = self.get_plugins()


        meta = {
            "artifacts_versions": artifacts_versions, 
            "hierarchy_info": hierarchy_info,
            "packaging": packaging,
            "config_server_availability": csa,
            "plugins": plugins
        }

        return meta


    def get_artifacts_versions(self):
        """
        
        """
        if self.pp:
            properties = self.pp.get_properties()
            result = {}
            for k,v in properties.items():
                if ".version" in k:
                    result[k] = v

            
            if "java.version" not in result:
                # we need to search further
                # not a springboot based  app
                java_version_mvn_compiler = self.pp.get_java_version_from_mvn_compiler()
                if java_version_mvn_compiler is not None:
                    result["java.version"] = java_version_mvn_compiler
                else:
                    # default value:
                    result["java.version"] = "1.8"

            return result
        else: 
            return {}

    def get_hierarchy(self):
        """

        """
        result = {"is_module": False}

        if self.pp:
            parent = self.pp.get_parent()  
            if parent == None:
                return result

            if "relativePath" in parent:
                
                if parent["relativePath"] != None:
                    relative_path = parent["relativePath"]

                    if relative_path == "../":
                        r_path = r_path = join("/".join(self.app_path.split("/")[:-1]),"pom.xml")
                        if isfile(r_path):
                            parent_pom = PomProcessor(r_path)
                            modules = parent_pom.get_modules()
                            if self.basename in modules:
                                result["is_module"] = True
                                result["parent_pom"] = r_path
                    else:
                        print("not supported")
                        # TODO: need to find a way to transform relative paths
                        
                else:
                    # assume "../"  
                    r_path = join("/".join(self.app_path.split("/")[:-1]),"pom.xml")
                    if isfile(r_path):
                        parent_pom = PomProcessor(r_path)
                        modules = parent_pom.get_modules()
                        if self.basename in modules:
                            result["is_module"] = True
                            result["parent_pom"] = r_path
            else:
                # assume "../"  

                r_path = join("/".join(self.app_path.split("/")[:-1]),"pom.xml")
                if isfile(r_path):
                    parent_pom = PomProcessor(r_path)
                    modules = parent_pom.get_modules()
                    if self.basename in modules:
                        result["is_module"] = True
                        result["parent_pom"] = r_path
        return result


    def get_config_server_availability(self):
        """

        """
        result = {"is_config_server_available" : False}

        hierarchy = self.get_hierarchy()
        if hierarchy["is_module"] == True:
            # lets get to the parent and get the list of modules
            # then, traverse them and check if any of them is a config server
            parent_pom = PomProcessor(hierarchy["parent_pom"])
            modules = parent_pom.get_modules()
            for module in modules: 
                
                # construct the path to the pom.xml
                candidate_pom_path = self.app_path.split("/")[:-1] +[module, "pom.xml"]
                candidate_pom_path = "/".join(candidate_pom_path)
                if isfile(candidate_pom_path):
                    # if it exists, lets see 
                    module_pom  = PomProcessor(candidate_pom_path)
                    dependencies = module_pom.get_dependencies()
                    for dependency in dependencies:
                        if "artifactId" in dependency:
                            if dependency["artifactId"] == "spring-cloud-config-server":
                                # found it

                                if module == self.basename:
                                    result["is_this_module_config_server"] = True
                                else:
                                    result["is_this_module_config_server"] = False

                                result["is_config_server_available"] = True
                                result["name"] = module
                                result["path"] = candidate_pom_path
        
        return result

    def get_packaging(self):
        """

        """
        result = { "packaging": ""}
        if self.pp:
            packaging = self.pp.get_packaging()
            if packaging is not None:
                result["packaging"] = packaging

        hierarchy = self.get_hierarchy()
        if hierarchy["is_module"] == True:
            # seach on the parent pom for "packaging"
            parent_pom = PomProcessor(hierarchy["parent_pom"])
            parent_pom_packaging = parent_pom.get_packaging()
            result["parent_pom_packaging"] = parent_pom_packaging

        return result

    def get_plugins(self):
        result = {"plugins" : [] }

        if self.pp:
            plugins = self.pp.get_plugins()
            result["plugins"] = plugins
        
        return result