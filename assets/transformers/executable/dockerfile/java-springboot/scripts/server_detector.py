import ipdb

from maven_processor import PomProcessor

class ServerDetector():
    """
    Application server detector for Java
    https://en.wikipedia.org/wiki/List_of_application_servers#Java
    """

    def __init__(self, captured_data, framework_params):
        self.captured_data = captured_data
        self.framework_params = framework_params

        if len(self.captured_data["maven_build_automation_files"]) > 0 :
            self.pom_path = self.captured_data["maven_build_automation_files"][0]
            self.pp = PomProcessor(self.pom_path) 

    def detect(self):
        """

        """
        if "springboot" in self.framework_params:

            result = {}
            # by default, we assume it is embedded
            result["is_embedded"] = True

            # analyze the pom
            if self.pp : 
                dependencies = self.pp.get_dependencies()
                
                # 1. search for a tomcat dependency with scope = provided
                for dependency in dependencies: 
                    if dependency["artifactId"] == "spring-boot-starter-tomcat":
                        result["server"] = "tomcat"
                     
                        if dependency["scope"] == "provided":
                            result["is_embedded"] = False
                # 2. the app has a liberty server enabled
                liberty_server_file = ""
                for i in self.captured_data["app_config_files"]:
                    if "server.xml" in i:
                        liberty_server_file = i 

                if liberty_server_file != "":
                    result["is_embedded"] = False

            return result
        else:

            # If no springboot , assume is_embedded == False 
            # TODO: add support for more frameworks
            return {"is_embedded":False}
