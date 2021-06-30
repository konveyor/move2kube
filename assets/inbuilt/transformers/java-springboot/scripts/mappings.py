import os
import json
import ipdb
from collections import defaultdict
from settings import CONSOLIDATED_IMAGE_SOURCE

def process_consolidated_images():
    
    dir_path = os.path.dirname(os.path.realpath(__file__))
    #ipdb.set_trace()
    with open(os.path.join(dir_path,"image_processor/output/consolidated_images.json"), "r") as f:
        consolidated_images = json.load(f)

    images_liberty = consolidated_images["liberty"]
    images_wildfly = consolidated_images["wildfly"]

    images_liberty = {k.replace(".json", ""):v for k,v in images_liberty.items()}
    images_wildfly = {k.replace(".json", ""):v for k,v in images_wildfly.items()}

    liberty2java, wildfly2java = {}, {}
    liberty2java = { k: v["java"] for k,v in images_liberty.items()}
    wildfly2java = { k: v["java"] for k,v in images_wildfly.items()}

    java2liberty = defaultdict(list)
    java2wildfly = defaultdict(list)

    for k,v in liberty2java.items():
        java2liberty[v["java"]].append(k)

    for k,v in wildfly2java.items():
        java2wildfly[v["java"]].append(k)

    return java2liberty, java2wildfly, liberty2java, wildfly2java


"""
Maps a Java version to a formal name
"""
java_versions_mapping = {

    # pom -> formal naming
    "11"    : "java-11-openjdk-devel",
    "1.8.0" : "java-1.8.0-openjdk-devel",
    "1.8"   : "java-1.8.0-openjdk-devel"
}

"""
Maps a Java version to a compatible Jboss image
"""
java_jboss_image_compatibility_mapping ={
    # pom -> formal image name 
    "11"   : "jboss/wildfly:18.0.0.Final",
    "1.8.0": ""
}

"""
Maps a Java version to a compatible Liberty image
"""
java_liberty_image_compatibility_mapping ={
    # pom -> formal image name 
    "11"   : "",
    "1.8": "openliberty/open-liberty:kernel-java8-openj9-ubi"
}



"""
List feasible paths when deploying Jboss apps
https://hub.docker.com/r/jboss/wildfly
"""
jboss_deployment_paths = [
    "/opt/jboss/wildfly/standalone/deployments/", 
    "/opt/jboss/wildfly/domain/deployments/"
]


java_images_mapping={
    "8": "openjdk:8",
    "11": "openjdk:11", 
    "1.8.0": "openjdk:8",
    "1.8": "openjdk:8",
    "1.7": "openjdk:7"
}