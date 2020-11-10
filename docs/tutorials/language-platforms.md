---
layout: default
title: "Language Platforms"
permalink: /tutorials/language-platforms/
parent: Tutorials
nav_order: 6
---

# Language Platforms

## Description

This document explains steps that will install move2kube and use 2 step process to containerize, and create deployment artifacts.

## Steps

1. Install Move2Kube `curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -`
1. Do `move2kube plan -s samples/language-platforms`.
1. Review `m2k.plan`
1. Do `move2kube translate`
1. Answer the questions and you will get the yaml files required for deploying the docker compose files in kubernetes.
