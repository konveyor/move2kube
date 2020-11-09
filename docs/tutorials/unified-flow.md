---
layout: default
title: "Unified flow"
permalink: /tutorials/unified-flow/
parent: Tutorials
nav_order: 8
---

# Unified Flow

## Description

This document explains steps that will install move2kube and use 3 step process to collect, containerize, and create deployment artifacts.

## Prerequisites

Install Move2Kube `curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -`

We will deploy a simple nodejs application into Cloud Foundry. If you have a CF app already you may use that instead.

1. Provision a CF app with the name `move2kube-demo-cf` using your cloud provider (Ex: IBM cloud). Make note of the API endpoint.
2. Login to cf using `cf login -a <YOUR CF API endpoint>`. If you don't have the CF CLI tool install that first from [here](https://docs.cloudfoundry.org/cf-cli/install-go-cli.html).
3. From the root folder of this repo run this to deploy the sample application `cd samples/unified-flow/cfnodejs; cf push`
4. Go to the URL of the application (you can get this by running `cf apps`) to see it running.

## Steps

1. From your terminal, login and set kubectl context.
1. Login to your cf instance
1. Do `move2kube collect`. This will create a `m2k_collect` folder, which will have files similar to `samples/unified-flow/cfapps.yaml` and `samples/unified-flow/cluster.yaml`. You can replace those files from your m2k_collect as appropriate.
1. Do `move2kube plan -s samples/unified-flow`
1. Review `m2k.plan`
1. Do `move2kube translate -c`
1. As you answer your questions, you will notice option to select target artifact type. Select among `Helm`, `Yamls` or `Knative`.
1. In the clusters, it will give an option to target the cluster you just collected above.
1. Answer the questions and you will get the helm chart/yaml files required for deploying your application in your kubernetes cluster.
