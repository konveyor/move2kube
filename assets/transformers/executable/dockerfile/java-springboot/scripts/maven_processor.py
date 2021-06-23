import xmltodict
import ipdb

class PomProcessor():

    def __init__(self, path_to_pom):
        self.path_to_pom = path_to_pom
        self.pom_dict = self.pom2dict()

    # Load the original pom.xml file and 
    # transform it into a python dictionary
    def pom2dict(self):
        with open(self.path_to_pom, "r") as f:
            pom_content = f.read()
            dict_obj = xmltodict.parse(pom_content)

        return dict_obj


    # Get the list of dependencies as a list
    def get_dependencies(self):
                           
        if "dependencies" in self.pom_dict["project"]:
            return [d for d in self.pom_dict["project"]["dependencies"]["dependency"]]
        else:
            return []

    # Get the application file that would be generated after building 
    # (and stored under the target/ folder)
    # Currently it is the contactenation of the name and version 
    def get_generated_app_file(self):

        project = self.pom_dict["project"]

        if "name" in project:
            name = project["name"]
        else:
            name = project["artifactId"]
            
        version = project["version"]
            
        nv = name + "-" +version
        if "packaging" in project:
            nv = nv + ".war"
        else: 
            nv = nv + ".jar"

        return nv

    def get_properties(self):
        project = self.pom_dict["project"]
        return project["properties"]

    def get_parent(self):

        project = self.pom_dict["project"]
        if "parent" in project:
            return project["parent"]
        else: 
            return None

    def get_modules(self):
        project = self.pom_dict["project"]
        if "modules" in project:
            return project["modules"]["module"]
        else:
            return []

    def get_packaging(self):
        project = self.pom_dict["project"]
        if "packaging" in project:
            return project["packaging"]
        else:
            return None

    def get_dependencies(self):
        project = self.pom_dict["project"]
        if "dependencies" in project:
            dependencies = project["dependencies"]
            return dependencies["dependency"]
        else:
            return []

    def get_plugins(self):
        project = self.pom_dict["project"]
        if "build" in project:
            build = project["build"]
            if "plugins" in build:
                return build["plugins"]
        else:
            return []


    def get_java_version_from_mvn_compiler(self):
        # https://www.baeldung.com/maven-java-version#compiler
        properties = self.get_properties()
        if "maven.compiler.source" in properties:
            return properties["maven.compiler.source"]
        elif "maven.compiler.release" in properties:
            return properties["maven.compiler.release"]
        else:
            # not found
            return None