[![Build](https://github.com/konveyor/move2kube/workflows/Build/badge.svg "Github Actions")](https://github.com/konveyor/move2kube/actions?query=workflow%3ABuild)
[![Container Repository on Quay](https://quay.io/repository/konveyor/move2kube/status "Container Repository on Quay")](https://quay.io/repository/konveyor/move2kube)
[![License](https://img.shields.io/:license-apache-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)
[![Go Report Card](https://goreportcard.com/badge/github.com/konveyor/move2kube)](https://goreportcard.com/report/github.com/konveyor/move2kube)
[<img src="https://img.shields.io/badge/slack-konveyor/move2kube-green.svg?logo=slack">](https://kubernetes.slack.com/archives/CR85S82A2)

# Move2Kube

Move2Kube is a command-line tool that accelerates the process of re-platforming to Kubernetes/Openshift. It does so by analyzing the environment and source artifacts, and asking guidance from the user when required. It allows customizations to enable generating the directory structure and artifacts in the format required for your project.

![Usage](./imgs/overview.png)

## Installation

### Using the install script

To install the latest stable version:

```shell
bash <(curl https://raw.githubusercontent.com/konveyor/move2kube/main/scripts/install.sh)
```

To install a specific version (for example version `v0.3.0-alpha.3`):

```shell
MOVE2KUBE_TAG='v0.3.0-alpha.3' bash <(curl https://raw.githubusercontent.com/konveyor/move2kube/main/scripts/install.sh)
```

To install the bleeding edge version:

```shell
BLEEDING_EDGE='true' bash <(curl https://raw.githubusercontent.com/konveyor/move2kube/main/scripts/install.sh)
```

### Uninstall CLI installed via the install script

Simply remove the binary

```shell
rm /usr/local/bin/move2kube
```

### Using Homebrew

```shell
brew tap konveyor/move2kube
brew install move2kube
```

### Uninstall CLI installed via Homebrew

```shell
brew uninstall move2kube
brew untap konveyor/move2kube
```

## UI

To bring up UI version:

Using `docker`:

```shell
docker run --rm -it -p 8080:8080 quay.io/konveyor/move2kube-ui:latest
```

Using `podman`:

```shell
podman run --rm -it -p 8080:8080 quay.io/konveyor/move2kube-ui:latest
```

Then go to http://localhost:8080 in a browser

More detailed instructions can be found in the [Move2Kube UI repo](https://github.com/konveyor/move2kube-ui#starting-the-ui)

## Usage

`move2kube transform -s src`, where `src` is the folder containing the source artifacts.

Checkout the [Tutorials](https://move2kube.konveyor.io/tutorials) and [Documentation](https://move2kube.konveyor.io/commands) for more information.

## Development environment setup

To browse code  [![Open in VSCode](https://badgen.net/badge/icon/Visual%20Studio%20Code?icon=visualstudio&label)](https://open.vscode.dev/konveyor/move2kube)

1. Obtain a recent version of `golang`. Known to work with `1.19`.
1. Ensure `$GOPATH` is set. If it's not set:
   1. `mkdir ~/go`
   1. `export GOPATH=~/go`
1. Obtain this repo:
   1. `mkdir -p $GOPATH/src/`
   1. Clone this repo into the above directory.
   1. `cd $GOPATH/src/move2kube`
1. Build: `make build`
1. Run unit tests: `make test`

## Artifacts Required

| Source | Artifact available | Features supported |
|:-------|:-------------------|:-------------------|
| Cloud Foundry | Manifest files, Source code | Containerization options based on source code, Deployment artifacts |
| Cloud Foundry | Manifest files, Source code, Access to running instance | Containerization options based on source code, Deployment artifacts, Metadata from runtime |
| Dockerfile | Dockerfile | Deployment artifacts, CI/CD pipeline artifacts |
| Docker Compose/Swarm | Docker compose files | Deployment artifacts |
| Docker Compose/Swarm | Docker compose files, Docker images | Deployment artifacts, CI/CD pipeline artifacts |
| Source Directories | Source code with no source metadata |  Containerization options based on source code, Deployment artifacts, CI/CD artifacts |
| Kubernetes Yamls | Kubernetes Yamls | Change versions, parameterize and create Helm chart, Kustomize yamls and Openshift templates. |

## Output

* Deployment artifacts
  * Dockerfile
  * Kubernetes/Openshift Yamls
  * Helm charts
  * Kustomize
  * OpenShift Templates
  * Docker compose

## Some Useful Configuration 
You can set an alias for move2kube to make it more convenient to use. The following command allows you to refer to move2kube as m2k for the current terminal session:

```
alias m2k="move2kube"

```
### To configure it globally: 
To keep aliases between sessions, you can save them in your user’s shell configuration profile file

#### Bash (.bashrc or .bash_profile)
```
echo 'alias m2k="move2kube"' >> ~/.bashrc
source ~/.bashrc

```

## Discussion

* For any questions reach out to us on any of the communication channels given on our website https://move2kube.konveyor.io/

## contributer 

You can click the Fork button in the upper-right area of the screen to create a copy of this repository in your GitHub account. This copy is called a fork. Make any changes you want in your fork, and when you are ready to send those changes to us, go to your fork and create a new pull request to let us know about it.

Once your pull request is created, a maintainer will take responsibility for providing clear, actionable feedback. As the owner of the pull request, it is your responsibility to modify your pull request to address the feedback that has been provided to you by the maintainer.

For more information about contributing , see:

 [contributing.md](https://github.com/konveyor/move2kube/blob/main/contributing.md)