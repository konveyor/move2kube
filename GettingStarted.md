---
layout: default
title: "Getting Started"
permalink: /getting-started
---

## Getting Started

There are two ways of running Move2Kube over the source artifacts.

### One step simple approach

##### Input
* `src/` – Directory containing the source artifacts. Copy the `docker-compose` folder from the [samples](https://github.com/konveyor/move2kube-demos/tree/main/samples/) and paste it inside the `src/` 

##### Run
```
$ move2kube translate -s src/
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
INFO[0002] [*source.Any2KubeTranslator] Done            
INFO[0002] Translation planning done                    
INFO[0002] Planning Metadata                            
INFO[0002] [*metadata.ClusterMDLoader] Planning metadata
INFO[0002] [*metadata.ClusterMDLoader] Done             
INFO[0002] [*metadata.K8sFilesLoader] Planning metadata
INFO[0002] [*metadata.K8sFilesLoader] Done              
INFO[0002] [*metadata.QACacheLoader] Planning metadata  
INFO[0002] [*metadata.QACacheLoader] Done               
INFO[0002] Metadata planning done
```

> Move2Kube asks for guidance from the user in form of questions-answers when required, to help in customizing the configurations for the specific cluster type the user is planning to deploy.

```
? 1. Which services should we expose?
Hints:
 [An Ingress object will be created for every exposed service.]
  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [✓]  redis
  [✓]  src
  [✓]  myproject-api
  [✓]  myproject-web
  [✓]  web
  [✓]  api

? 2. Choose the cluster type:
Hints:
 [Choose the cluster type you would like to target]
  [Use arrows to move, type to filter]
  Azure-AKS
  GCP-GKE
  IBM-IKS
  IBM-Openshift
> Kubernetes
  Openshift
  AWS-EKS

? 3. Choose the artifact type:
Hints:
 [Yamls - Generate Kubernetes Yamls Helm - Generate Helm chart Knative - Create Knative artifacts]
  [Use arrows to move, type to filter]
> Yamls
  Helm
  Knative

? 4. Select the registry where your images are hosted:
Hints:
 [You can always change it later by changing the yamls.]
  [Use arrows to move, type to filter]
  Other
  index.docker.io
> docker.io


INFO[0003] Customization done                            
INFO[0003] Execution completed                          
INFO[0003] Translated target artifacts can be found at [myproject].
```

##### Output
* `myproject/` - Directory containing the target deployment artifacts like
  * Containerization scripts
    * Dockerfile
    * Source 2 Image (S2I)
    * Cloud Native Buildpack
  * Deployment artifacts
    * Kubernetes/Openshift Yamls
    * Helm charts
    * Operator
    * Docker compose


### Two step involved approach

1. **Plan** : Place source code in a directory say `src` and generate a *plan*. For example, you can use the `docker-compose` folder from the [samples](https://github.com/konveyor/move2kube-demos/tree/main/samples/).

```
$ move2kube plan -s src/
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
INFO[0004] [*source.Any2KubeTranslator] Done            
INFO[0004] Translation planning done                    
INFO[0004] Planning Metadata                            
INFO[0004] [*metadata.ClusterMDLoader] Planning metadata
INFO[0004] [*metadata.ClusterMDLoader] Done             
INFO[0004] [*metadata.K8sFilesLoader] Planning metadata
INFO[0004] [*metadata.K8sFilesLoader] Done              
INFO[0004] [*metadata.QACacheLoader] Planning metadata  
INFO[0004] [*metadata.QACacheLoader] Done               
INFO[0004] Metadata planning done                       
INFO[0004] Plan can be found at [m2k.plan].
```
Generates a *plan* file containing a transformation proposal (including containerization options) for all the services discovered from various sources.

2. **Translate** : In the same directory, invoke the below command.
```
$ move2kube translate
```

Note: If information about any runtime instance say cloud foundry or kubernetes cluster needs to be collected, use `move2kube collect`. You can place the collected data in the src directory used in the plan. (Optional)
