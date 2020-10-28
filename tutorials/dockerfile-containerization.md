---
layout: default
title: "Dockerfile Containerization"
permalink: /tutorials/dockerfile-containerization.md/
---

# Dockerfile Containerization

## Description

This document helps you to add customized docker files which could be parameterized using scripts which could detect your application, extract any parameters from it and supply them to Move2Kube so that docker file with appropriate parameters is generated for your application gets created. We use `samples/language-platforms/java-maven` as a running example to illustrate the steps.

## Steps

1. Install Move2Kube `curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -`
2. Writing a dockerfile template:
    - To get a feel for the templates, check out other existing dockerfile containerizers [here](https://github.com/konveyor/move2kube/tree/master/internal/assets/dockerfiles).
    - Specifically, let us discuss [java-maven docker file](https://github.com/konveyor/move2kube/blob/master/internal/assets/dockerfiles/javamaven/Dockerfile) as an illustration. This docker file template has two sections: 
        - First one is the `build app` section where-in the base-layer and subsequent packages to be installed to facilitate the building of the `java-maven` app is specified. In this particular example, we have parameterized the `APPNAME`, but any other line could also be parameterized based on what we extract from the source artefacts (check out m2kdetect script section).
        - Second is the `run app` section where in the base-layer for running the app built from the previous section is specified, and the runnable artefacts obtained from the first phase are copied into this second image. All necessary dependencies for running the artefact could also be installed here. Note here that we have parameterized the port to be exposed. We could parameterize anything else as per the requirement of the language-platform.
        - Not all language-platforms might require two stage (build and run) dockerfile, but it is a good practice to separate it out as illustrated above.
3. Writing the `m2kdfdetect.sh`:
    - This script is supposed to perform the following functions:
        - Detect the language-platform
        - Extract the required parameters from the source artefacts so as to fill the dockerfile template created previously (E.g. APPNAME, Port).
    - A typical example of the detect script is shown [here](https://github.com/konveyor/move2kube/blob/master/internal/assets/dockerfiles/python/m2kdfdetect.sh).
    - The script has to return the paramaters in json format if the matching language-platform is detected. If not, it should exit with exit code `1`. Following is an illustration for the `java-maven` case:
    ```
    echo '{"Port": 8080, "APPNAME": "app"}'
    ```
4. Copy your `Dockerfile` and `m2kdfdetect.sh` to your application source folder (e.g. `samples/language-platforms/java-maven`).
5. Generate and test: 
    - Do `move2kube plan -s <srcfolder>; move2kube translate` (e.g. `<srcfolder>` is `samples/language-platforms/java-maven` or your sample application).
    - Answer the questions for customization and you will get in the output directory (e.g `myproject`):
        - Dockerfiles required to create the containers.
        - Shell scripts to build these containers.
        - Yaml files required for deploying these containers in kubernetes.
    - In the output directory, do `sh copysources.sh samples/language-platforms/java-maven` to copy sources for which container has to be created.
    - Do `sh buildimages.sh` to build the container. Once the image is built, `docker images` should list the image `java-maven:latest`.
    - If there are no bugs, then the image could be run with `docker run` command.