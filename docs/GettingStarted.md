---
layout: default
title: "Getting Started"
permalink: /getting-started
nav_order: 3
---

# Getting Started

There are two usage modes of Move2Kube:
* [Command Line Tool](#command-line-tool)
* [Web Interface](#web-interface)

## Command Line Tool

The Move2Kube command line tool can be used to generate the kubernetes deployment artifacts for the given source platform artifacts. There are two ways of running the Move2Kube command line tool over the source artifacts.

### One step simple approach
![One step usage of Move2Kube]({{ site.baseurl }}/assets/images/one-step-usage.png)

  ```console
  $ move2kube translate -s src/
  ```
Here, `src/` is the directory containing the source artifacts.

### Involved approach
![Involved usage of Move2Kube]({{ site.baseurl }}/assets/images/usage.png)

  ```console
  $ move2kube collect (optional)
  $ move2kube plan -s src/
  $ move2kube translate
  ```

## Web Interface
Move2Kube Web Interface takes as input the source artifacts in a zip file and generates the plan file and the target platform deployment artifacts.

![One step usage of Move2Kube]({{ site.baseurl }}/assets/images/m2k-ui.png)


Please refer the [Tutorials]({{ site.baseurl }}{% link docs/tutorials/Tutorial.md %}) for more detailed usage of Move2Kube.
