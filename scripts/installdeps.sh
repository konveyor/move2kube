#!/usr/bin/env bash

#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# initOS discovers the operating system for this system.

MOVE2KUBE_DEP_INSTALL_PATH="$PWD/bin"

initOS() {
    OS="$(uname | tr '[:upper:]' '[:lower:]')"

    case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows' ;;
    esac
}

askBeforeProceeding() {
    echo 'Move2Kube dependency installation script.'
    echo 'The following dependencies will be installed to '"$MOVE2KUBE_DEP_INSTALL_PATH"':'
    echo 'docker (only on Linux), pack, kubectl, operator-sdk'
    echo "$MOVE2KUBE_DEP_INSTALL_PATH"' will be added to the $PATH'
    read -r -p 'Proceed? [y/N]:' oktoproceed
}

# MAIN

initOS

if [ "$OS" == "darwin" ]; then
    PACKPLATFORM='macos'
    KUBECTLPLATFORM='darwin'
    OPERATORSDKPLATFORM='apple-darwin'
    echo 'Once this installation finishes, please do install docker from https://docs.docker.com/docker-for-mac/install/'
elif [ "$OS" == "linux" ]; then
    PACKPLATFORM='linux'
    KUBECTLPLATFORM='linux'
    OPERATORSDKPLATFORM='linux-gnu'
else
    echo 'Unsupported platform:'"$OS"' . Exiting.'
    exit 1
fi

askBeforeProceeding

if [ "$oktoproceed" != 'y' ]; then
    echo 'Failed to get confirmation. Exiting.'
    exit 1
fi

if [ "$OS" == "linux" ]; then
    if ! grep -q container_t '/proc/1/attr/current'; then
        curl -fsSL 'https://get.docker.com' -o 'get-docker.sh' && sudo sh 'get-docker.sh' && rm 'get-docker.sh'
    fi
fi

mkdir -p "$MOVE2KUBE_DEP_INSTALL_PATH"
curl -o pack.tgz -LJO 'https://github.com/buildpacks/pack/releases/download/v0.12.0/pack-v0.12.0-'"$PACKPLATFORM"'.tgz' && tar -xzf pack.tgz && mv pack "$MOVE2KUBE_DEP_INSTALL_PATH" && rm pack.tgz
curl -o kubectl -LJO 'https://storage.googleapis.com/kubernetes-release/release/'"$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)"'/bin/'"$KUBECTLPLATFORM"'/amd64/kubectl' && mv kubectl "$MOVE2KUBE_DEP_INSTALL_PATH"
chmod +x "$MOVE2KUBE_DEP_INSTALL_PATH"/kubectl
curl -o operator-sdk -LJO 'https://github.com/operator-framework/operator-sdk/releases/download/v1.0.0/operator-sdk-v1.0.0-x86_64-'"$OPERATORSDKPLATFORM" && mv operator-sdk "$MOVE2KUBE_DEP_INSTALL_PATH"
chmod +x "$MOVE2KUBE_DEP_INSTALL_PATH"/operator-sdk
echo 'PATH="$PATH:'"$MOVE2KUBE_DEP_INSTALL_PATH"'"' >>~/.bash_profile
echo 'Installed the dependencies to '"$MOVE2KUBE_DEP_INSTALL_PATH"
echo 'We have added a line to your '~/.bash_profile' to put '"$MOVE2KUBE_DEP_INSTALL_PATH"' on your $PATH. Either restart the shell or source ~/.bash_profile to see the changes.'
