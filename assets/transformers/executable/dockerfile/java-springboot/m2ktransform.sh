#!/usr/bin/env bash
#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

root_path=$(pwd)
basename=$(basename $1)
output_path="/tmp/"$basename

# remove any previous output 
rm -rf $output_path
mkdir -p $output_path

# app (server) specific configuration files
app_config_files=$output_path"/app_config_files.output"

# 
application_properties_files=$output_path"/application_properties_files.output"

# yamls
application_yml_files=$output_path"/application_yml_files.output"
yml_files=$output_path"/yml_files.output"


# maven or gradle files:
maven_build_automation_files=$output_path"/maven_build_automation_files.output"
gradle_build_automation_files=$output_path"/gradle_build_automation_files.output"
# in case of gradle, we save the output of the dependency tree
gradle_dependencies=$output_path"/gradle_dependencies.output"

# in case we have a .ear file, we decompress to this path
extracted_content_path=$output_path"/extracted_content/"

input_type=""

if [[ -d $1 ]]; then
    #echo "$1 is a directory"
	# setting content folder variable
	content_folder=$1
	input_type="directory"

	# check if the folder contains a java project
	# to the a java project, at least one .java file must exist
	JAVA_FILES=(`find $content_folder -type f -name "*.java"`)

	#printf '%s\n' "${JAVA_FILES[@]}"
	#echo ${#JAVA_FILES[@]}
	if [ ${#JAVA_FILES[@]} -eq 0 ]; then
		#echo "no java files, exiting"
		exit 1
	fi

elif [[ -f $1 ]]; then
    #echo "$1 is a file"
	input_type="file"

	# check if file is ear/war/jar
	file_extension="${basename##*.}"
	arr=('war' 'jar' 'ear')
	if [[ " ${arr[*]} " != *"$file_extension"* ]];
	then
		#echo "not jar/war/ear file. exiting"
		exit 1
	fi

	#echo "Extracting file contents to "$extracted_content_path
	rm -rf $extracted_content_path
	mkdir $extracted_content_path
	unzip -d $extracted_content_path ${1}
	cd $extracted_content_path
	FILES_TO_PROCESS=`find . -type f \( -name "*.ear" -or -name "*.war" -or -name "*.jar" \)`

	until [ "$FILES_TO_PROCESS" == "" ] ; do
		for myFile in $FILES_TO_PROCESS ; do
			mkdir ${myFile}-contents
			unzip -d ${myFile}-contents ${myFile}
			rm $myFile
		done
		FILES_TO_PROCESS=`find . -type f \( -name "*.ear" -or -name "*.war" -or -name "*.jar" \)`
	done
  
    # setting content folder variable
	content_folder=$extracted_content_path
else
    exit 1
fi

cd $root_path
#echo "Searching for build automation files"
find $content_folder -type f -name "*.gradle" > $gradle_build_automation_files
find $content_folder -type f -name "pom.xml" > $maven_build_automation_files

#echo "Searching for application configuration  xml files"
find $content_folder -type f -name "*.xml" > $app_config_files

#echo "Searching for application.properties files "
find $content_folder -type f -name "application.properties" > $application_properties_files

#echo "Searching for application.yml files (spring-boot)"
find $content_folder -type f \( -name "application.yml" -o -name "application.yaml" \) > $application_yml_files

# echo "Searching for all yaml files ( cover cloudfoundry)
find $content_folder -type f \( -name "*.yml" -o -name "*.yaml" \) > $yml_files


# check if gradle is present
gradle_lines=$(find $content_folder -type f -name "*.gradle" | wc -l | xargs)
#echo $gradle_lines
if [[ $gradle_lines  -gt 0 ]];then
	#echo "Gradle found. Obtaining gradle dependency tree"
	# get depdencies 
	chmod -R 755 $content_folder
    cd $content_folder
	gradle -q dependencies > $gradle_dependencies
	# go back to root path 
	cd $root_path
fi

# calling python 
abspath=$(realpath $content_folder)
python3 scripts/run.py --mode transform --app_path $(echo $abspath) --output_path $(echo $output_path) --basename $(echo $basename) --input_type $(echo $input_type)
