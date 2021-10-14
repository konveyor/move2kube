---
layout: default
title: "Web Interface"
permalink: /installation/web-interface/
parent: Installation
nav_order: 2
---

# Move2Kube Web Interface

## Bringing up Move2Kube all-in-one container

Choose the version ready for you use case:

**Stable Release** ready for production cases:

To run locally using `docker`:

```shell
$ mkdir -p workspace 
$ cd workspace
$ docker run -p 8080:8080 -v "${PWD}:/workspace" -v /var/run/docker.sock:/var/run/docker.sock --rm -it quay.io/konveyor/move2kube-aio:release-0.2
```

To run locally using `podman`:

```shell
$ mkdir -p workspace 
$ cd workspace
$ podman run -p 8080:8080 -v "${PWD}:/workspace:z" --rm -it quay.io/konveyor/move2kube-aio:release-0.2
```

**Latest** for development and testing cases:

To run locally using `docker`:

```shell
$ mkdir -p workspace
$ cd workspace
$ docker run -p 8080:8080 -v "${PWD}:/workspace" -v /var/run/docker.sock:/var/run/docker.sock --rm -it quay.io/konveyor/move2kube-aio:latest
```

To run locally using `podman`:

```shell
$ mkdir -p workspace
$ cd workspace
$ podman run -p 8080:8080 -v "${PWD}:/workspace:z" --rm -it quay.io/konveyor/move2kube-aio:latest
```

Access the UI in `http://localhost:8080/`.

   > Note: There is a known issue when using the above command in WSL.  
   The CNB containerization option will not be available.  
   Also an empty folder called `workspace` may be created in the root directory.  
   If you are on Windows, use Powershell instead of WSL until this is fixed.

## Bringing up Move2Kube UI and API as separate containers

Cloning the git repository:

```shell
$ git clone https://github.com/konveyor/move2kube-ui
$ cd move2kube-ui
$ mkdir -p data
```

The `data` folder will be used to persist the data managed by the container.

Start the container using `docker-compose`:

```shell
$ docker-compose up
```

Or using `podman-compose`:

```shell
$ podman-compose -f podman-compose.yml up
```

Access the UI in `http://localhost:8080/`.

## Bringing up Move2Kube UI as Helm Chart  

Move2Kube can also be installed as a Helm Chart from [ArtifactHub](https://artifacthub.io/packages/helm/move2kube/move2kube/0.2.0-beta.0?modal=install)

Also, for Helm Chart and Operator checkout [Move2Kube Operator](https://github.com/konveyor/move2kube-operator).
