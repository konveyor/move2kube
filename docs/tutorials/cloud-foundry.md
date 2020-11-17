---
layout: default
title: "Cloud Foundry"
permalink: /tutorials/cloud-foundry/
parent: Tutorials
nav_order: 3
---

# Cloud Foundry

## Description

This document explains steps that will install move2kube and use 3 step process to collect, containerize, and create deployment artifacts for cloud foundry apps. It also takes through the process to customize for a specific cluster.

## Prerequisites

Install Move2Kube `curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -`

We will deploy a simple nodejs application into Cloud Foundry. If you have a CF app already you may use that instead.

1. Provision a CF app with the name `move2kube-demo-cf` using your cloud provider (Ex: IBM cloud). Make note of the API endpoint.
2. Login to cf using `cf login -a <YOUR CF API endpoint>`. If you don't have the CF CLI tool install that first from [here](https://docs.cloudfoundry.org/cf-cli/install-go-cli.html).
3. From the root folder of this repo run this to deploy the sample application `cd samples/cloud-foundry/; cf push`
4. Go to the URL of the application (you can get this by running `cf apps`) to see it running.

## Steps

Now that we have a running Cloud Foundry app we can translate it using move2kube. Run these steps from the `samples/` folder:

1. We will first collect some data about your running CF application and any kubernetes clusters `move2kube collect -a cf`
2. The data we collected will be stored in a new folder called `m2k_collect`. Move this into the source directory `mv m2k_collect/ cloud-foundry/m2k_collect`
3. Then we create a plan on how to translate your app to run kubernetes `move2kube plan -s cloud-foundry/`
4. The plan is stored in the YAML file `m2k.plan`. Using the plan we created we can do the translation `move2kube translate`
5. Answer any questions that it may ask. If you don't know the answer just press Enter to select the default.

You should now have a folder called `myproject` containing the translated kubernetes artifacts.
