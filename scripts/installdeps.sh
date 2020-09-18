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

if [ "$(uname)" == "Darwin" ]; then
    PACKPLATFORM="macos"
    KUBECTLPLATFORM="darwin"
    OPERATORSDKPLATFORM="apple-darwin"
    echo "Once this installation finishes, please do install docker from https://docs.docker.com/docker-for-mac/install/"
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    PACKPLATFORM="linux"
    KUBECTLPLATFORM="linux"
    OPERATORSDKPLATFORM="linux-gnu"
    if ! grep -q container_t "/proc/1/attr/current"; then
        curl -fsSL get.docker.com -o get-docker.sh && sudo sh get-docker.sh && rm get-docker.sh
    fi
else
    echo "Unsupported platform."
    exit 1
fi

mkdir -p bin
curl -o pack.tgz -LJO https://github.com/buildpacks/pack/releases/download/v0.12.0/pack-v0.12.0-$PACKPLATFORM.tgz && tar -xzf pack.tgz && mv pack bin/ && rm pack.tgz
curl -o kubectl -LJO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/$KUBECTLPLATFORM/amd64/kubectl && mv kubectl bin/
chmod +x bin/kubectl
curl -o operator-sdk -LJO https://github.com/operator-framework/operator-sdk/releases/download/v1.0.0/operator-sdk-v1.0.0-x86_64-$OPERATORSDKPLATFORM && mv operator-sdk bin/
chmod +x bin/operator-sdk
echo "PATH=$PATH:$PWD/bin" >> ~/.bash_profile
source ~/.bash_profile
