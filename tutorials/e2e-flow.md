---
layout: default
title: "End-to-End Flow"
permalink: /tutorials/e2e-flow/
---


# End-to-End Flow of using Move2Kube

## Description

This document explains the steps that will install Move2Kube and also tells how to use Move2Kube to containerize and create deployment artifacts. We will be using the data from [samples/e2e-flow](https://github.com/konveyor/move2kube-demos/tree/main/samples/e2e-flow) which has two applications, one golang appication and one nodejs application. These applications can be Cloud Foundry applications or they can be normal applications deployed to VMs. Let's see the structure inside the `samples/e2e-flow` directory.

![e2e-flow-directory-sturcture](../../images/samples/e2e-flow/e2e-flow-tree.png)

Now, let's see how these applications can be translatred to Kubernetes. Here we are going to use the one step process for this sample usecase.

## Prerequisites

1. Install Move2Kube

   ```
   $ curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -
   ```

2. Install [operator-sdk](https://docs.openshift.com/container-platform/4.1/applications/operator_sdk/osdk-getting-started.html#osdk-installing-cli_osdk-getting-started). To verify that operator-sdk was correctly installed

   ```
   $ operator-sdk version
   ```

3. Clone the [move2kube-demos](https://github.com/konveyor/move2kube-demos) repository

   ```
   $ git clone https://github.com/konveyor/move2kube-demos.git
   
   $ cd move2kube-demos
   ```

## Steps to generate target artifacts

<span>1.</span> Execute

```
$ move2kube translate -s samples/e2e-flow
```

![Plan creation](../../images/samples/e2e-flow/planning.png)

* It goes through each and every file and tries to analyze and understand each one of them. Then it tries to interact with you whenever it has a doubt. It creates the *plan* for you internally and then will come back to you when it has some doubts.

![Question 1](../../images/samples/e2e-flow/ques1.png)

* It has identified two services (golang and nodejs), and is asking if you want to translate all of them.

![Question 2](../../images/samples/e2e-flow/ques2.png)

* In both the services, Move2Kube can translate using multiple containerization techniques. It might be Dockerfile, S2I (Source-To-Image) or CNB (Cloud Native Buildpack). Let's just go with the Dockerfile.

![Question 3](../../images/samples/e2e-flow/ques3.png)

* Now it asks, what do you want to create - Yaml files, Helm charts or Knative artifacts? Let's go with the Helm.

![Question 4](../../images/samples/e2e-flow/ques4.png)

* What kind of clustering you are going to deploy to (OpenShift or Kubernetes or particular flavors of Kubernetes)? Here we select Kubernetes.

![Question 5](../../images/samples/e2e-flow/ques5.png)

* What are the services that you want to expose externally? Going with the by-default (all of them).

![Question 6](../../images/samples/e2e-flow/ques6.png)

* Then it asks to select the registry where your images are hosted. Select 'Other' if your registry name is not here.

![Question 7](../../images/samples/e2e-flow/ques7.png)

* Enter the name of the registry where you host the images. Here we enter 'us.icr.io' registry name.

![Question 8](../../images/samples/e2e-flow/ques8.png)

* Input the namespace under which you want to deploy- m2kdemo.

![Question 9](../../images/samples/e2e-flow/ques9.png)

* Now it asks about the type of container registry login.

![Question 10](../../images/samples/e2e-flow/ques10.png)

* Then, it asks about the name of the pull secret.

![Translation Done!](../../images/samples/e2e-flow/translation-complete.png)

Finally, the translation is successful and the target artifacts can be found inside the `myproject` folder. The structure of the *myproject* folder can be seen by executing the below command.

```
$ tree myproject
```
![tree myproject](../../images/samples/e2e-flow/tree-myproject.png)

The created Helm charts are stored inside the *myproject/myproject* directory. The Readme.md file guides on the next steps to be followed. Many scripts like buildimages.sh, copysources.sh, helminstall.sh and docker-compose.yaml are also present inside the *myproject* folder. Move2Kube also created a Helm-based operator for you inside the *myproject/myproject-operator*. Next step will be to deploy the applications using the created target artifacts.
