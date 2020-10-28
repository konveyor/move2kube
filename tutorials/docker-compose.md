---
layout: default
title: "Docker compose translation"
permalink: /tutorials/docker-compose/
---

# Docker compose translation

## Description

This document explains steps that will install move2kube and translate docker-compose artifacts.

## Steps

1. Install Move2Kube `curl -L https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh | bash -`
1. Do `move2kube translate -s samples/docker-compose`.
1. Answer the questions and you will get the yaml files required for deploying the docker compose files in kubernetes.
