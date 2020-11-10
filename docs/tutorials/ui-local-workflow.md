---
layout: default
title: "UI Local Machine Workflow"
permalink: /tutorials/ui-local-workflow/
parent: Tutorials
nav_order: 7
---

# UI Local Machine Workflow

## Description

This document explains steps to bring up UI and backend using docker-compose and use it for translation.

## Steps

1. Download the latest `docker-compose.yml` file from `https://raw.githubusercontent.com/konveyor/move2kube-ui/master/docker-compose.yml`
1. Install `docker-compose`, if you don't have it locally yet.
1. Create a folder by the name `wksps` in the same folder as the docker compose file
1. Do a `docker-compose up`
1. Create a new project
1. Click on the three dots in the end and select `details`
1. Click on the `assets` tab and upload the `samples/ui-local-workflow.zip` file
1. Go to the `Plan` tab
1. Click on the `Generate Plan` button
1. @ait for the plan to get generated. It takes about a min or two.
1. Click on `Refresh` in the plan tab
1. Review the plan
1. Go to `Artifacts` tab
1. Click on `Translate` button
1. Answer the questions as apt
1. Download the generated artifacts, extract it and browse them
