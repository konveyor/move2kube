import re
import ipdb
import json
import time
import argparse

from os.path import join, isfile
from os import listdir
from collections import defaultdict
from pprint import pprint



# separator token
sep_token = "_sep_"

# image name -> output path 
images = {
    "wildfly":  "/Users/pablo/Desktop/image_processor_output/jboss_sep_wildfly",
    "liberty":  "/Users/pablo/Desktop/image_processor_output/openliberty_sep_open-liberty"
}

java_tokens  = ["java" , "JAVA_VERSION"]
java_filter_tokens = ["JAVA_HOME"]

def get_files_in_folder(path):
    return [f for f in listdir(path) if isfile(join(path, f))]

def process_images():
    result = {}

    result_image2config = {}


    for image, image_path in images.items(): 
        print("processing", image)
        result[image] = {}

        result_image2config[image] = {}

        image_list  = get_files_in_folder(image_path)
        print(" ", len(image_list), "files found")

        for im in image_list:
            print(" ", im)

            try:
                with open(join(image_path, im), "r") as f:
                    im_data = json.load(f)

                # created attribute
                if "created" in im_data:
                    created = im_data["created"]
                else: 
                    created = ""
            
                # history attribute
                # why use history?
                # https://medium.com/@saschagrunert/demystifying-containers-part-iii-container-images-244865de6fef

                if "history" in im_data:
                    im_data_history = im_data["history"]
                else:
                    im_data_history = []

                # contains candidate tokens across commands of  the history
                # these tokens are likely to have the java version within
                cmd_candidate_tokens = []

                for h in im_data_history:
                    if "created_by" in h:
                        cmd = h["created_by"]

                        cmd_tokens  = cmd.split(" ")
                        for ct in cmd_tokens:
                            if any([t in ct for t in java_tokens]) is True and any([t in ct for t in java_filter_tokens]) is False:
                                cmd_candidate_tokens.append(ct)
        
                java_version = get_java_version(cmd_candidate_tokens)
                #print(java_version)

                result[image][im] = {
                    "created" : created, 
                    "java" : java_version
                }

                if "config" in im_data:
                    result_image2config[image][im] = im_data["config"]

            except:
                print(" ->>could not open file")

    with open("output/consolidated_images.json", "w") as f:
        json.dump(result, f) 

    with open("output/consolidated_images2config.json", "w") as f:
        json.dump(result_image2config, f) 


def numericalize_java_version(example):

    # https://www.oracle.com/java/technologies/javase/jdk8-naming.html
    if example.isnumeric():
        return example
    else:
        if "jdk8u" in example:
            return "1.8"
        else:
            return example


def get_java_version(cmd_candidate_tokens):

    # cascading 
    # case 1. give priority if any of the tokens have  a JAVA_VERSION= substring
    java_version_token = "JAVA_VERSION="

    for t in cmd_candidate_tokens:
        if java_version_token in t: 
            candidate_value = t.split("=")[1]


            if candidate_value is not None:
                candidate_value = numericalize_java_version(candidate_value)
                return {"java": candidate_value}
    

    # case 2. no java version found we need to check the remaining 
    
    # main assumption 
    # versions follow a key-value form
    # eg :java-11-openjdk-devel
    # java : 11
    # openjdk : devel

    # delimiters
    tokens = ["java" , "openjdk" , "jdk"]
    
    # iterate over candidate tokens
    for example in cmd_candidate_tokens:
        t2i = {}
        for t in tokens:
            try:
                if t == "jdk" and "openjdk" in t2i:
                    ne = example.replace("openjdk", "")
                    t2i[t] = ne.index(t)
                t2i[t] = example.index(t)
            except:
                pass

        sorted_t2i = sorted(list(t2i.items()), key= lambda x : x[1] )  

        t2segment  = {}
        for s, f in zip(sorted_t2i, sorted_t2i[1:]):
            t2segment[s[0]] = (s[1], f[1])
        
        t2segment[sorted_t2i[-1][0]] =  (sorted_t2i[-1][1], len(example)) 

        t2param = {}
        for t , (s,f) in t2segment.items():
            param = example[s + len(t): f].replace("-","")
            t2param[t]= param

        return t2param

def main(args):
    process_images()


if __name__ == "__main__":

    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", type=str, default="all")
    args = parser.parse_args()
    main(args)