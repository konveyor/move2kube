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

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.

[[ $DEBUG ]] || DEBUG='false'
[[ $BINARY_NAME ]] || BINARY_NAME='move2kube'
[[ $TAG ]] || TAG='v0.1.0-alpha'
[[ $USE_SUDO ]] || USE_SUDO='true'
[[ $VERIFY_CHECKSUM ]] || VERIFY_CHECKSUM='true'
[[ $MOVE2KUBE_INSTALL_DIR ]] || MOVE2KUBE_INSTALL_DIR='/usr/local/bin'

HAS_CURL="$(type curl &>/dev/null && echo true || echo false)"
HAS_WGET="$(type wget &>/dev/null && echo true || echo false)"
HAS_OPENSSL="$(type openssl &>/dev/null && echo true || echo false)"
HAS_SHA256SUM="$(type sha256sum &>/dev/null && echo true || echo false)"

initArch() {
    ARCH="$(uname -m)"
    case $ARCH in
    armv5*) ARCH="armv5" ;;
    armv6*) ARCH="armv6" ;;
    armv7*) ARCH="arm" ;;
    aarch64) ARCH="arm64" ;;
    x86) ARCH="386" ;;
    x86_64) ARCH="amd64" ;;
    i686) ARCH="386" ;;
    i386) ARCH="386" ;;
    esac
}

# initOS discovers the operating system for this system.
initOS() {
    OS=$(uname | tr '[:upper:]' '[:lower:]')

    case "$OS" in
    # Minimalist GNU for Windows
    mingw*) OS='windows' ;;
    esac
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
verifySupported() {
    local supported="darwin-amd64\nlinux-amd64"
    if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
        echo "No prebuilt binary for ${OS}-${ARCH}."
        echo "To build from source, go to https://github.com/konveyor/move2kube"
        exit 1
    fi

    if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
        echo "Either curl or wget is required"
        exit 1
    fi

    if [ "${VERIFY_CHECKSUM}" == "true" ] && [ "${HAS_OPENSSL}" != "true" ] && [ "${HAS_SHA256SUM}" != "true" ]; then
        echo "In order to verify checksum, sha256sum or openssl must first be installed."
        echo "Please install sha256sum or openssl or set VERIFY_CHECKSUM=false in your environment."
        exit 1
    fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
    MOVE2KUBE_DIST="move2kube-$TAG-$OS-$ARCH.zip"
    DOWNLOAD_URL="https://github.com/konveyor/move2kube/releases/download/$TAG/$MOVE2KUBE_DIST"
    CHECKSUM_URL="$DOWNLOAD_URL.sha256sum"
    MOVE2KUBE_TMP_ROOT="$(mktemp -dt move2kube-installer-XXXXXX)"
    MOVE2KUBE_TMP_FILE="$MOVE2KUBE_TMP_ROOT/$MOVE2KUBE_DIST"
    MOVE2KUBE_SUM_FILE="$MOVE2KUBE_TMP_ROOT/$MOVE2KUBE_DIST.sha256sum"

    echo "Downloading $DOWNLOAD_URL"
    if [ "${HAS_CURL}" == "true" ]; then
        curl -o "$MOVE2KUBE_SUM_FILE" -sSL "$CHECKSUM_URL"
        curl -o "$MOVE2KUBE_TMP_FILE" -sSL "$DOWNLOAD_URL"
    elif [ "${HAS_WGET}" == "true" ]; then
        wget -q -O "$MOVE2KUBE_SUM_FILE" "$CHECKSUM_URL"
        wget -q -O "$MOVE2KUBE_TMP_FILE" "$DOWNLOAD_URL"
    fi
}

# verifyFile verifies the SHA256 checksum of the binary package
# and the GPG signatures for both the package and checksum file
# (depending on settings in environment).
verifyFile() {
    if [ "${VERIFY_CHECKSUM}" == "true" ]; then
        verifyChecksum
    fi
}

# verifyChecksum verifies the SHA256 checksum of the binary package.
verifyChecksum() {
    printf "Verifying checksum... "

    local expected_sum
    expected_sum="$(awk '{print $1}' <"${MOVE2KUBE_SUM_FILE}")"

    local sum=0
    if [ "$HAS_SHA256SUM" == "true" ]; then
        sum="$(sha256sum "${MOVE2KUBE_TMP_FILE}" | awk '{print $1}')"
    else
        sum="$(openssl sha1 -sha256 "${MOVE2KUBE_TMP_FILE}" | awk '{print $2}')"
    fi

    if [ "$sum" != "$expected_sum" ]; then
        echo "SHA sum of ${MOVE2KUBE_TMP_FILE} does not match. Aborting."
        exit 1
    fi
    echo "Done."
}

# installFile installs the move2kube binary.
installFile() {
    MOVE2KUBE_TMP="$MOVE2KUBE_TMP_ROOT/$BINARY_NAME"
    mkdir -p "$MOVE2KUBE_TMP"
    bsdtar -xf "$MOVE2KUBE_TMP_FILE" -C "$MOVE2KUBE_TMP"
    MOVE2KUBE_TMP_BIN="$MOVE2KUBE_TMP/$OS-$ARCH/$BINARY_NAME"
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
        echo "$BINARY_NAME not found. Is $MOVE2KUBE_INSTALL_DIR on your "'$PATH?'
        exit 1
    fi
    set -e
}

# cleanup temporary files.
cleanup() {
    if [[ -d "${MOVE2KUBE_TMP_ROOT:-}" ]]; then
        rm -rf "$MOVE2KUBE_TMP_ROOT"
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
    downloadFile
    verifyFile
    installFile
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
