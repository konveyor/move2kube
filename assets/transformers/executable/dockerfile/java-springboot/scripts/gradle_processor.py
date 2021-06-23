import xmltodict
import ipdb

class GradleProcessor():

    def __init__(self, path_to_gradle):
        self.path_to_gradle= path_to_gradle
        self.gradle_dict = self.gradle2dict()

    def gradle2dict(self):
        return {}