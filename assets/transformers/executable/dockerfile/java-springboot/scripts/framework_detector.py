import ipdb
from maven_processor import PomProcessor


class FrameworkDetector():
    """
    Tries to detect the main framework the target application is built on 
    """

    def __init__(self, captured_data, application_path):
        self.captured_data = captured_data
        self.application_path = application_path
    
    def detect(self):

        rule_based_results = {
            "springboot" : self.contains_springboot()
        }

        # Do any of these rule-based function found something?
        rule_based_success = any([ v is True for k,v in rule_based_results.items()])

        if rule_based_success:
            return rule_based_results
        else:
            heuristic_based_results =  self.detect_heuristic()
            return heuristic_based_results

    def contains_springboot(self, dependency_key="artifactId"):

        # Rule 1: we need to have spring boot components as dependencies
        r1 = "maven_build_automation_files" in self.captured_data and len(self.captured_data["maven_build_automation_files"]) > 0
        has_sp_deps = False

        if r1:
            pom_path = self.captured_data["maven_build_automation_files"][0] # todo: choose among multiple 
            pp = PomProcessor(pom_path)
            dependencies = pp.get_dependencies()
            has_sp_deps  = any(["spring-boot" in d[dependency_key] for d in dependencies])
        

        # Rule 2: we need to have an `application.properties` or `application.yaml/.yml`
        r2 =  ("application_yml_files" in self.captured_data and len(self.captured_data["application_yml_files"]) > 0 ) or (
            "application_properties_files" in self.captured_data and len(self.captured_data["application_properties_files"]) > 0)

        if has_sp_deps or r2:
            return True
        else:
            return False

    def detect_heuristic(self):

        return {
            "springboot" : False,
            "liberty"    : False
        }
