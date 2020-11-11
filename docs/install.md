---
layout: default
title: "Installation"
permalink: /installation
nav_order: 3
---

## Installation

### Move2Kube Command Line Tool:

**Linux / macOS / Windows WSL (Recommended):**
   ```console
   $ bash <(curl https://raw.githubusercontent.com/konveyor/move2kube/master/scripts/install.sh)
   ```

### Move2Kube Web Interface:

   ```console
   $ git clone https://github.com/konveyor/move2kube-ui
   $ docker-compose up
   ```

   Move2Kube UI will now be accessible at http://localhost:8080.
<br>

### Other alternate ways of installing Move2Kube:

**Github release**

The binary can be downloaded from the [GitHub releases page](https://github.com/konveyor/move2kube/releases) of Move2Kube.

**Go**

Installing using `go get` pulls from the master branch of [Move2Kube](https://github.com/konveyor/move2kube) with the latest development changes.
   ```console
   $ go get –u github.com/konveyor/move2kube
   ```
