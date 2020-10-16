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

[[ $DEBUG ]] || DEBUG='false'

MOVE2KUBE_DEP_INSTALL_PATH="$PWD/bin"

HAS_DOCKER="$(type docker &>/dev/null && echo true || echo false)"
HAS_PACK="$(type pack &>/dev/null && echo true || echo false)"
HAS_KUBECTL="$(type kubectl &>/dev/null && echo true || echo false)"
HAS_OPERATOR_SDK="$(type operator-sdk &>/dev/null && echo true || echo false)"
if [ "$HAS_OPERATOR_SDK" == 'true' ]; then
    if operator-sdk version | cut -d, -f1 | grep 'operator-sdk version: "v1' -q; then
        OPERATOR_SDK_V1='true'
    else
        OPERATOR_SDK_V1='false'
    fi
fi

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
    read -r -p 'Proceed? [y/N]:' oktoproceed
}

askIfWeShouldModifyBashProfile() {
    read -r -p 'Should we add a line to your ~/.bash_profile that will append '"$MOVE2KUBE_DEP_INSTALL_PATH"' to the $PATH? [y/N]:' oktoaddtobashprofile
}

# fail_trap is executed if an error occurs.
fail_trap() {
    result=$?
    if [ "$result" != "0" ]; then
        echo "Failed to install the dependencies."
        echo -e "\tFor support, go to https://github.com/konveyor/move2kube"
    fi
    exit $result
}

# MAIN

#Stop execution on any error
trap "fail_trap" EXIT
set -e
set -u

# Set debug if desired
if [ "${DEBUG}" == "true" ]; then
    set -x
fi

initOS

if [ "$OS" == 'darwin' ]; then
    PACKPLATFORM='macos'
    KUBECTLPLATFORM='darwin'
    OPERATORSDKPLATFORM='apple-darwin'
    echo 'Once this installation finishes, please do install docker from https://docs.docker.com/docker-for-mac/install/'
elif [ "$OS" == 'linux' ]; then
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

mkdir -p "$MOVE2KUBE_DEP_INSTALL_PATH"

if [ "${HAS_DOCKER}" != 'true' ] && [ "$OS" == 'linux' ]; then
    if ! grep -q container_t '/proc/1/attr/current'; then
        echo 'Installing docker...'
        curl -fsSL 'https://get.docker.com' -o 'get-docker.sh' && sudo sh 'get-docker.sh' && rm 'get-docker.sh'
        echo 'Done.'
    fi
fi

if [ "${HAS_PACK}" != 'true' ]; then
    echo 'Installing pack...'
    curl -o pack.tgz -LJO 'https://github.com/buildpacks/pack/releases/download/v0.12.0/pack-v0.12.0-'"$PACKPLATFORM"'.tgz' && tar -xzf pack.tgz && mv pack "$MOVE2KUBE_DEP_INSTALL_PATH" && rm pack.tgz
    echo 'Done.'
fi

if [ "${HAS_KUBECTL}" != 'true' ]; then
    echo 'Installing kubectl...'
    curl -o kubectl -LJO 'https://storage.googleapis.com/kubernetes-release/release/'"$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)"'/bin/'"$KUBECTLPLATFORM"'/amd64/kubectl' && mv kubectl "$MOVE2KUBE_DEP_INSTALL_PATH"
    chmod +x "$MOVE2KUBE_DEP_INSTALL_PATH"/kubectl
    echo 'Done.'
fi

if [ "${HAS_OPERATOR_SDK}" != 'true' ] || [ "$OPERATOR_SDK_V1" != 'true' ]; then
    echo 'Installing operator-sdk...'
    curl -o operator-sdk -LJO 'https://github.com/operator-framework/operator-sdk/releases/download/v1.0.0/operator-sdk-v1.0.0-x86_64-'"$OPERATORSDKPLATFORM" && mv operator-sdk "$MOVE2KUBE_DEP_INSTALL_PATH"
    chmod +x "$MOVE2KUBE_DEP_INSTALL_PATH"/operator-sdk
    echo 'Done.'
fi

echo 'Installed the dependencies to '"$MOVE2KUBE_DEP_INSTALL_PATH"

# Check if $MOVE2KUBE_DEP_INSTALL_PATH is already in the $PATH
if [[ :"$PATH": == *:"$MOVE2KUBE_DEP_INSTALL_PATH":* ]]; then
    echo "$MOVE2KUBE_DEP_INSTALL_PATH"' is already in $PATH'
else
    askIfWeShouldModifyBashProfile
    if [ "$oktoaddtobashprofile" != 'y' ]; then
        echo 'Failed to get confirmation. Not modifying ~/.bash_profile'
    else
        echo 'PATH="$PATH:'"$MOVE2KUBE_DEP_INSTALL_PATH"'"' >>~/.bash_profile
        echo 'We have added a line to your ~/.bash_profile to put '"$MOVE2KUBE_DEP_INSTALL_PATH"' on your $PATH. Either restart the shell or source ~/.bash_profile to see the changes.'
    fi
fi

echo 'Finished!!'
