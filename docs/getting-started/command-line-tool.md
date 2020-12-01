---
layout: default
title: "Command Line Tool"
permalink: /getting-started/cli/
parent: Getting Started
nav_order: 1
---

# Command Line Tool

The Move2Kube command line tool can be used to generate the kubernetes deployment artifacts for the given source platform artifacts. There are two ways of running the Move2Kube command line tool over the source artifacts.

## One step simple approach
![One step usage of Move2Kube]({{ site.baseurl }}/assets/images/one-step-usage.png)

  ```console
  $ move2kube translate -s src/
  ```
Here, `src/` is the directory containing the source artifacts.

## Involved approach
![Involved usage of Move2Kube]({{ site.baseurl }}/assets/images/usage.png)

  ```console
  $ move2kube collect (optional)
  $ move2kube plan -s src/
  $ move2kube translate
  ```

For more details about Move2Kube Command Line Tool:
* [Installation]({{ site.baseurl }}{% link docs/installation/command-line-tool.md %})
* [Tutorials]({{ site.baseurl }}{% link docs/tutorials/Tutorial.md %})
