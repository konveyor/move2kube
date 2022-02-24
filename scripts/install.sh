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

[[ $DEBUG ]] || DEBUG='false'

print_usage() {
    echo "Invalid args: $*"
    echo 'Usage: install.sh [-y]'
    echo 'Use sudo when running in -y quiet mode.'
}

QUIET=false
if [ "$#" -gt 0 ]; then
    if [ "$#" -gt 1 ] || [ "$1" != '-y' ]; then
        print_usage "$@"
        exit 1
    fi
    echo 'Installing without prompting. Script should be run with sudo in order to install to /usr/local/bin'
    QUIET=true
fi

[[ $USE_SUDO ]] || USE_SUDO='true'
[[ $BINARY_NAME ]] || BINARY_NAME='move2kube'
[[ $MOVE2KUBE_TAG ]] || MOVE2KUBE_TAG='latest'
[[ $BLEEDING_EDGE ]] || BLEEDING_EDGE='false'
[[ $VERIFY_CHECKSUM ]] || VERIFY_CHECKSUM='true'
[[ $MOVE2KUBE_INSTALL_DIR ]] || MOVE2KUBE_INSTALL_DIR='/usr/local/bin'

HAS_JQ="$(command -v jq >/dev/null && echo true || echo false)"
HAS_SUDO="$(command -v sudo >/dev/null && echo true || echo false)"
HAS_CURL="$(command -v curl >/dev/null && echo true || echo false)"
HAS_WGET="$(command -v wget >/dev/null && echo true || echo false)"
HAS_OPENSSL="$(command -v openssl >/dev/null && echo true || echo false)"
HAS_SHA256SUM="$(command -v sha256sum >/dev/null && echo true || echo false)"
HAS_MOVE2KUBE="$(command -v "$BINARY_NAME" >/dev/null && echo true || echo false)"

if [ "$USE_SUDO" = "true" ] && [ "$HAS_SUDO" = 'false' ]; then
    echo 'executable "sudo" not found. Proceeding without sudo, some commands may fail to execute properly.'
    USE_SUDO='false'
fi

isURLExist() {
    if [ "$#" -ne 1 ]; then
        echo 'isURLExist needs exactly 1 arg: the url to check'
        echo "actual args: $*"
        exit 1
    fi
    if [ "$HAS_CURL" = 'true' ]; then
        curl -fsS --head "$1" >/dev/null 2>&1
        return
    fi
    wget -q --spider "$1"
}

download() {
    if [ "$#" -ne 2 ]; then
        echo 'download needs exactly 2 args: the url to download and the output path'
        echo "actual args: $*"
        exit 1
    fi
    echo "Downloading $1"
    if [ "$HAS_CURL" = 'true' ]; then
        curl -fsSL -o "$2" "$1"
    else
        wget -q -O "$2" "$1"
    fi
}

# verifyChecksum verifies the SHA256 checksum of the binary package.
verifyChecksum() {
    if [ "$#" -ne 2 ]; then
        echo 'verifyChecksum needs exactly 2 args: the path to the file and the path to the checksum file'
        echo "actual args: $*"
        exit 1
    fi
    echo 'Verifying checksum'
    local expected_sum
    expected_sum="$(awk '{print $1}' <"$2")"
    local sum=''
    if [ "$HAS_SHA256SUM" = 'true' ]; then
        sum="$(sha256sum "$1" | awk '{print $1}')"
    else
        sum="$(openssl sha1 -sha256 "$1" | awk '{print $2}')"
    fi
    if [ "$sum" != "$expected_sum" ]; then
        echo "SHA sum of $1 does not match. Aborting."
        exit 1
    fi
}

downloadAndVerifyChecksum() {
    if [ "$#" -ne 2 ]; then
        echo 'downloadAndVerifyChecksum needs exactly 2 args: the url to download and the output path'
        echo "actual args: $*"
        exit 1
    fi
    download "$1" "$2"
    local checksum_url="$1"'.sha256sum'
    local checksum_path="$2"'.sha256sum'
    download "$checksum_url" "$checksum_path"
    verifyChecksum "$2" "$checksum_path"
    rm "$checksum_path"
}

initArch() {
    ARCH="$(uname -m)"
    case $ARCH in
    armv5*) ARCH='armv5' ;;
    armv6*) ARCH='armv6' ;;
    armv7*) ARCH='arm' ;;
    aarch64) ARCH='arm64' ;;
    x86) ARCH='386' ;;
    x86_64) ARCH='amd64' ;;
    i686) ARCH='386' ;;
    i386) ARCH='386' ;;
    esac
}

# initOS discovers the operating system for this system.
initOS() {
    OS="$(uname | tr '[:upper:]' '[:lower:]')"
    case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows' ;;
    esac
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
verifySupported() {
    local supported='darwin-amd64\nlinux-amd64'
    if ! echo "$supported" | grep -q "${OS}-${ARCH}"; then
        echo "No prebuilt binary for ${OS}-${ARCH}."
        echo "To build from source, go to https://github.com/konveyor/move2kube"
        exit 1
    fi
    if [ "$HAS_CURL" != 'true' ] && [ "$HAS_WGET" != 'true' ]; then
        echo 'Either curl or wget is required'
        exit 1
    fi
    if [ "$VERIFY_CHECKSUM" = 'true' ] && [ "$HAS_OPENSSL" != 'true' ] && [ "$HAS_SHA256SUM" != 'true' ]; then
        echo 'In order to verify checksum, sha256sum or openssl must first be installed.'
        echo 'Please install sha256sum or openssl or set VERIFY_CHECKSUM=false in your environment.'
        exit 1
    fi
    installDependencies
}

# installDependencies installs all the dependencies we need.
installDependencies() {
    JQ='jq'
    if [ "$HAS_JQ" = 'false' ]; then
        JQ="$(mktemp -d)/jq"
        if [ "${OS}-${ARCH}" = 'darwin-amd64' ]; then
            download 'https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64' "$JQ"
        else
            download 'https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64' "$JQ"
        fi
        chmod +x "$JQ"
    fi
}

# getLatestVersion gets the latest release version.
getLatestVersion() {
    # Get tag from releaseinfo.json
    local json_data=''
    local release_info_url='https://move2kube.konveyor.io/releaseinfo.json'
    if [ "$HAS_CURL" = 'true' ]; then
        json_data="$(curl -fsSL "$release_info_url")"
    elif [ "$HAS_WGET" = 'true' ]; then
        json_data="$(wget -qO - "$release_info_url")"
    fi
    if [ "$BLEEDING_EDGE" != 'true' ]; then
        echo 'installing the latest stable version'
        MOVE2KUBE_TAG="$(printf '%s\n' "$json_data" | "$JQ" -r '.current.release')"
        return
    fi
    echo 'installing the bleeding edge version'
    MOVE2KUBE_TAG="$(printf '%s\n' "$json_data" | "$JQ" -r '.next_next.prerelease')"
    if [ "$MOVE2KUBE_TAG" != 'null' ]; then
        return
    fi
    MOVE2KUBE_TAG="$(printf '%s\n' "$json_data" | "$JQ" -r '.next.prerelease')"
    if [ "$MOVE2KUBE_TAG" != 'null' ]; then
        return
    fi
    local current_prerelease
    local current_release
    current_prerelease="$(printf '%s\n' "$json_data" | "$JQ" -r '.current.prerelease')"
    current_release="$(printf '%s\n' "$json_data" | "$JQ" -r '.current.release')"
    if [[ "$current_prerelease" = "$current_release"* ]]; then
        MOVE2KUBE_TAG="$(printf '%s\n' "$json_data" | "$JQ" -r '.current.release')"
        return
    fi
    MOVE2KUBE_TAG="$(printf '%s\n' "$json_data" | "$JQ" -r '.current.prerelease')"
}

# checkMove2KubeInstalledVersion checks which version of move2kube is installed and
# if it needs to be changed.
checkMove2KubeInstalledVersion() {
    if [ "$HAS_MOVE2KUBE" = 'true' ]; then
        local version
        version="$("$BINARY_NAME" version)"
        if [ "$version" = "$MOVE2KUBE_TAG" ]; then
            echo "The desired Move2Kube version $version is already installed"
            return 0
        else
            echo "Move2Kube $MOVE2KUBE_TAG is available. Changing from version $version"
            return 1
        fi
    else
        return 1
    fi
}

# downloadMove2Kube downloads the latest binary package and verifies the checksum.
downloadMove2Kube() {
    MOVE2KUBE_DIST="move2kube-$MOVE2KUBE_TAG-$OS-$ARCH.tar.gz"
    DOWNLOAD_URL="https://github.com/konveyor/move2kube/releases/download/$MOVE2KUBE_TAG/$MOVE2KUBE_DIST"
    MOVE2KUBE_TMP_ROOT="$(mktemp -dt move2kube-installer-XXXXXX)"
    MOVE2KUBE_TMP_FILE="$MOVE2KUBE_TMP_ROOT/$MOVE2KUBE_DIST"
    if [ "$VERIFY_CHECKSUM" = 'true' ]; then
        downloadAndVerifyChecksum "$DOWNLOAD_URL" "$MOVE2KUBE_TMP_FILE"
    else
        download "$DOWNLOAD_URL" "$MOVE2KUBE_TMP_FILE"
    fi
}

# installMove2Kube installs the move2kube binary.
installMove2Kube() {
    MOVE2KUBE_TMP="$MOVE2KUBE_TMP_ROOT/$BINARY_NAME"
    mkdir -p "$MOVE2KUBE_TMP"
    tar -xf "$MOVE2KUBE_TMP_FILE" -C "$MOVE2KUBE_TMP"
    MOVE2KUBE_TMP_BIN="$MOVE2KUBE_TMP/$BINARY_NAME/$BINARY_NAME"
    echo "Preparing to install $BINARY_NAME into ${MOVE2KUBE_INSTALL_DIR}"
    runAsRoot cp "$MOVE2KUBE_TMP_BIN" "$MOVE2KUBE_INSTALL_DIR/$BINARY_NAME"
    echo "Successfully installed $BINARY_NAME into $MOVE2KUBE_INSTALL_DIR/$BINARY_NAME"
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
    local CMD="$*"
    if [ "$EUID" != "0" ] && [ "$USE_SUDO" == "true" ]; then
        CMD="sudo $CMD"
    fi
    $CMD
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
    set +e
    if ! command -v "$BINARY_NAME" >/dev/null; then
        echo "$BINARY_NAME not found. Is $MOVE2KUBE_INSTALL_DIR on your PATH?"
        exit 1
    fi
    set -e
}

getc() {
    local save_state
    save_state=$(/bin/stty -g)
    /bin/stty raw -echo
    IFS= read -r -n 1 -d '' "$@"
    /bin/stty "$save_state"
}

wait_for_user() {
    local c
    echo
    echo "Press RETURN to continue or any other key to skip"
    getc c
    # we test for \r and \n because some stuff does \r instead
    if ! [[ "$c" == $'\r' || "$c" == $'\n' ]]; then
        return 1
    fi
    return 0
}

# cleanup temporary files.
cleanup() {
    echo 'cleaning up temporary files and directories...'
    if [[ -d "${MOVE2KUBE_TMP_ROOT:-}" ]]; then
        rm -rf "$MOVE2KUBE_TMP_ROOT"
    fi
    if [ "$JQ" != 'jq' ]; then
        rm -rf "$(dirname "$JQ")"
    fi
}

# fail_trap is executed if an error occurs.
fail_trap() {
    result=$?
    if [ "$result" != "0" ]; then
        echo "Failed to install $BINARY_NAME"
        echo -e "\tFor support, go to https://github.com/konveyor/move2kube"
    fi
    cleanup
    exit $result
}

main() {
    echo 'Installing move2kube'
    initArch
    initOS
    verifySupported
    if [ "$MOVE2KUBE_TAG" = 'latest' ]; then
        getLatestVersion
    fi
    if ! checkMove2KubeInstalledVersion; then
        downloadMove2Kube
        installMove2Kube
    fi
    testVersion
    echo 'Done!'
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

main
