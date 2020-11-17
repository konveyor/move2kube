---
layout: default
title: "Docker compose translation"
permalink: /tutorials/docker-compose/
parent: Tutorials
nav_order: 2
---

# Docker compose translation

## Description

This document explains steps that will install move2kube and translate docker-compose artifacts. Let's try to take a Docker compose file and deploy it to Kubernetes. We will be using the data from [samples/docker-compose](https://github.com/konveyor/move2kube-demos/tree/main/samples/docker-compose).

## Prerequisites

1. Install Move2Kube.

   ```console
   $ bash <(curl https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh)
   ```

2. Install dependencies.
  * [Docker](https://www.docker.com/get-started)
  * [operator-sdk](https://docs.openshift.com/container-platform/4.1/applications/operator_sdk/osdk-getting-started.html#osdk-installing-cli_osdk-getting-started)
  * [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
  * [Pack](https://buildpacks.io/docs/tools/pack/)

   For convenience, we have provided a script which can help you to install all these dependencies in one go.

   ```console
   $ bash <(curl https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/installdeps.sh)
   ```
   To verify that dependencies were correctly installed
   ```console
   $ operator-sdk version
   ```
   ```console
   $ docker version
   ```
   ```console
   $ pack version
   ```
   ```console
   $ kubectl version
   ```
3. Install [Helm v3](https://helm.sh/docs/intro/install/)

   To verify that Helm v3 was correctly installed
   ```console
   $ helm version
   ```

4. Clone the [move2kube-demos](https://github.com/konveyor/move2kube-demos) repository

   ```console
   $ git clone https://github.com/konveyor/move2kube-demos.git
   ```

   ```console
   $ cd move2kube-demos
   ```
   Let's see the structure inside the `samples/docker-compose` directory.

   ```console
   move2kube-demos git:(main) $ tree samples/docker-compose
   samples/docker-compose
   └── docker-compose.yaml
   ```

   Here we have a single docker-compose file inside the source directory, but the it could have multiple docker-compose files also. Move2Kube has the capability to go through all the docker-compose files and combine them and give a holistic view for you.

## Steps to generate target artifacts

<span>1.</span> Execute `move2kube translate -s samples/docker-compose`.

```console

$ move2kube-demos git:(main) ✗ move2kube translate -s samples/docker-compose
INFO[0000] Planning Translation                         
INFO[0000] [*source.DockerfileTranslator] Planning translation
INFO[0000] [*source.DockerfileTranslator] Done          
INFO[0000] [*source.ComposeTranslator] Planning translation
INFO[0000] [*source.ComposeTranslator] Done             
INFO[0000] [*source.CfManifestTranslator] Planning translation
INFO[0000] [*source.CfManifestTranslator] Done          
INFO[0000] [*source.KnativeTranslator] Planning translation
INFO[0000] [*source.KnativeTranslator] Done             
INFO[0000] [*source.KubeTranslator] Planning translation
INFO[0000] [*source.KubeTranslator] Done                
INFO[0000] [*source.Any2KubeTranslator] Planning translation
INFO[0007] [*source.Any2KubeTranslator] Done            
INFO[0007] Translation planning done                    
INFO[0007] Planning Metadata                            
INFO[0007] [*metadata.ClusterMDLoader] Planning metadata
INFO[0007] [*metadata.ClusterMDLoader] Done             
INFO[0007] [*metadata.K8sFilesLoader] Planning metadata
INFO[0007] [*metadata.K8sFilesLoader] Done              
INFO[0007] [*metadata.QACacheLoader] Planning metadata  
INFO[0007] [*metadata.QACacheLoader] Done               
INFO[0007] Metadata planning done
```

* It goes through each and every file and tries to analyze and understand each one of them. Then it tries to interact with you whenever it has a doubt. It creates the *plan* for you internally and then will come back to you when it has some doubts.

```console
? 1. Select all services that are needed:
Hints:
 [The services unselected here will be ignored.]
  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [✓]  web
  [✓]  api
  [✓]  redis
```

It has found three services- api service, redis service and web service. So, it is asking whether you want to translate all three of them? Here, we select all three services.

```console
? 2. Select all containerization modes that is of interest:
Hints:
 [The services which does not support any of the containerization technique you are interested will be ignored.]
  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [✓]  Reuse
```

Do you want to reuse the container images? Here, we go ahead with the default which is to reuse.

```console
? 3. Choose the artifact type:
Hints:
 [Yamls - Generate Kubernetes Yamls Helm - Generate Helm chart Knative - Create Knative artifacts]
  [Use arrows to move, type to filter]
> Yamls
  Helm
  Knative
```

Whether you want Helm charts, Yamls or Knative artifacts? Let's go ahead with Yamls.

```console
? 4. Choose the cluster type:
Hints:
 [Choose the cluster type you would like to target]
  [Use arrows to move, type to filter]
  AWS-EKS
  Azure-AKS
  GCP-GKE
  IBM-IKS
  IBM-Openshift
> Kubernetes
  Openshift
```

Now, it asks to select the cluster type you want to deploy to. We will deploy to Kubernetes cluster.

```console
? 5. Select all services that should be exposed:
Hints:
 [The services unselected here will not be exposed.]
  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
  [ ]  redis
  [✓]  web
> [ ]  api

INFO[1303] Optimization done                            
INFO[1303] Begin Customization
```

Select the services which needs to be exposed. We want to expose the web service.

Now we will go ahead with the default specifications for everything else (by pressing the return key).

```console
? 6. [] What type of container registry login do you want to use?
Hints:
 [Docker login from config mode, will use the default config from your local machine.]
  [Use arrows to move, type to filter]
  Use existing pull secret
> No authentication
  UserName/Password

? 7. Provide the ingress host domain
Hints:
 [Ingress host domain is part of service URL]
 myproject.com

? 8. Provide the TLS secret for ingress
 Hints:
  [Enter TLS secret name]

INFO[1369] Customization done                           
INFO[1369] Execution completed                          
INFO[1369] Translated target artifacts can be found at [myproject].   
```

Finally, the translation is successful and the target artifacts can be found inside the *myproject* folder. The structure of the *myproject* folder can be seen by executing the below command.

```console
move2kube-demos git:(main) $ tree myproject
myproject
├── Readme.md
├── containers
├── deploy.sh
├── docker-compose.yaml
├── m2kqacache.yaml
└── myproject
    ├── api-deployment.yaml
    ├── api-service.yaml
    ├── myproject-ingress.yaml
    ├── redis-deployment.yaml
    ├── redis-service.yaml
    ├── web-deployment.yaml
    └── web-service.yaml
```

Since this is a pre-containerized environment, container files are already there and only the Kubernetes artifacts are created. Move2Kube has created deployment artifacts, service yamls and ingress for the different services. So, this is a quick way where you can take your docker-compose file and within few seconds you can have all your Kubernetes artifacts required to deploy to your cluster.
