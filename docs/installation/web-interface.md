---
layout: default
title: "Web Interface"
permalink: /installation/web-interface/
parent: Installation
nav_order: 2
---

# Move2Kube Web Interface

## Bringing up Move2Kube all-in-one container

   ```console
   $ mkdir -p workspace && cd workspace
   $ docker run -p 8080:8080 -v "${PWD}:/workspace" -v /var/run/docker.sock:/var/run/docker.sock --rm -it quay.io/konveyor/move2kube-aio:latest
   ```
   Access the UI in `http://localhost:8080/`.

   > Note: There is a known issue when using the above command in WSL.  
   The CNB containerization option will not be availabe.  
   Also an empty folder called `workspace` may be created in the root directory.  
   If you are on Windows, use Powershell instead of WSL until this is fixed.

## Bringing up Move2Kube UI and API as separate containers

   ```console
   $ git clone https://github.com/konveyor/move2kube-ui
   $ cd move2kube-ui
   $ docker-compose up
   ```
   Access the UI in `http://localhost:8080/`.

For Helm chart and Operator checkout [Move2Kube Operator](https://github.com/konveyor/move2kube-operator).

<br>
