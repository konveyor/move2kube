from collections import defaultdict
from os import listdir
from os.path import isfile, join

class DataLoader():

    def __init__(self, output_path):
        self.output_path = output_path

    def load_captured_data(self):

        """
        This function standarizes the list of files captured by the bash script. 
        There are 5 categories:
            - app_config_files : any xml file found
            - application_properties_files: any application.properties found
            - application_yml_files: any application.yml or application.yaml found
            - gradle_build_automation_files : any *.gradle found
            - maven_build_automation_files: any pom.xml found (could have overlap with app_config_files)
            - gradle_dependencies: not a file, but the dumped output of gradle  
        """

        data = defaultdict(list)
        data["build_type"] = "undefined"


        if isfile(join(self.output_path, "app_config_files.output")):
            with open(join(self.output_path, "app_config_files.output"), "r") as f:
                data_app_config_files = [l.strip() for l in f.readlines()]
            if len(data_app_config_files) > 0:
                data["app_config_files"] = data_app_config_files


        if isfile(join(self.output_path, "application_properties_files.output")):
            with open(join(self.output_path, "application_properties_files.output"), "r") as f:
                data_application_properties_files = [l.strip() for l in f.readlines()]
            if len(data_application_properties_files) > 0:
                data["application_properties_files"] = data_application_properties_files


        if isfile(join(self.output_path, "application_yml_files.output")):
            with open(join(self.output_path, "application_yml_files.output"), "r") as f:
                data_application_yml_files = [l.strip() for l in f.readlines()]
            if len(data_application_yml_files) > 0:
                data["application_yml_files"] = data_application_yml_files


        if isfile(join(self.output_path, "yml_files.output")):
            with open(join(self.output_path, "yml_files.output"), "r") as f:
                data_yml_files = [l.strip() for l in f.readlines()]
            if len(data_yml_files) > 0:
                data["yml_files"] = data_yml_files
        
        if isfile(join(self.output_path, "gradle_build_automation_files.output")):
            with open(join(self.output_path, "gradle_build_automation_files.output"), "r") as f:
                data_gradle_build_automation_files = [l.strip() for l in f.readlines()]     
            if len(data_gradle_build_automation_files) > 0:
                data["gradle_build_automation_files"] = data_gradle_build_automation_files
                data["build_type"] = "gradle"


        if isfile(join(self.output_path, "maven_build_automation_files.output")):
            with open(join(self.output_path, "maven_build_automation_files.output"), "r") as f:
                data_maven_build_automation_files = [l.strip() for l in f.readlines()]     
            if len(data_maven_build_automation_files) > 0:
                data["maven_build_automation_files"] = data_maven_build_automation_files
                data["build_type"] = "maven"


        if isfile(join(self.output_path, "gradle_dependencies.output")):
            with open(join(self.output_path, "gradle_dependencies.output"), "r") as f:
                data["gradle_dependencies"] = [l.strip() for l in f.readlines()]
                
        return data