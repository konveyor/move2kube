---
layout: default
title: "Kubernetes-to-Kubernetes"
permalink: /tutorials/kubernetes-to-kubernetes/
parent: Tutorials
nav_order: 5
---

# Kubernetes-to-Kubernetes

## Description

This document explains steps that will install move2kube and use a single step process to convert yamls to fit a specific cluster version or flavor like openshift.

## Steps

1. Install Move2Kube `curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -`
1. Do a `move2kube translate -s samples/kubernetes-to-kubernetes`
1. Answer the questions, in the cluster section, choose `Openshift` instead of `Kubernetes`
1. Answer all remaining questions
1. The artifacts generated will be Openshift artifacts, and if you had selected helm chart, it will be a helm chart as output.

If you want to target a specific custom cluster type, use `move2kube collect` once the cluster is logged in for kubectl, and place the `m2k_collect` folder inside the input folder for `move2kube translate`. This will give additional option, in addition to `kubernetes` and `openshift` and move2kube will create artifacts with kinds supported for that specific cluster.
