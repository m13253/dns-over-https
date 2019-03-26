#!/bin/bash 
set -euo pipefail

# This is a script used for automated packaging.
# Debian maintainers please don't use this.
# 
# Environment assumption:
#  * Ubuntu 16.04
#  * run with normal user
#  * sudo with no password
#  * go and fpm is pre-installed
#  * rpmbuild is required if you need rpm packages

export DEBIAN_FRONTEND="noninteractive"
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export BUILD_BINARIESDIRECTORY="${BUILD_BINARIESDIRECTORY:-${DIR}/build/bin}"
export BUILD_ARTIFACTSTAGINGDIRECTORY="${BUILD_ARTIFACTSTAGINGDIRECTORY:-${DIR}/build/packages}"
export TMP_DIRECTORY="/tmp/dohbuild"
export GOPATH="${GOPATH:-/tmp/go}"
export GOBIN="${GOBIN:-/tmp/go/bin}"

function prepare_env() {
    echo "Checking dependencies"

    if ! [ -x "$(command -v go)" ]; then
        echo "Please install golang"
        exit 1
    fi

    if [ -x "$(command -v apt-get)" ]; then
        sudo apt-get -y update
    fi

    if ! [ -x "$(command -v rpmbuild)" ]; then
        # TODO: currectly install rpmbuild
        ! sudo apt-get -y install rpmbuild
    fi

    # if ! [ -x "$(command -v upx)" ]; then
    #     sudo apt-get -y install upx
    # fi

    echo "Creating directories"

    mkdir -p "${BUILD_BINARIESDIRECTORY}/nm-dispatcher"
    mkdir -p "${BUILD_BINARIESDIRECTORY}/launchd"
    mkdir -p "${BUILD_BINARIESDIRECTORY}/systemd"
    mkdir -p "${BUILD_BINARIESDIRECTORY}/config"
    mkdir -p "${BUILD_ARTIFACTSTAGINGDIRECTORY}"
    mkdir -p "${TMP_DIRECTORY}"
}

function build_common() {
    cp NetworkManager/dispatcher.d/* "${BUILD_BINARIESDIRECTORY}"/nm-dispatcher
    cp launchd/*.plist "${BUILD_BINARIESDIRECTORY}"/launchd
    cp systemd/*.service "${BUILD_BINARIESDIRECTORY}"/systemd
    cp doh-server/doh-server.conf "${BUILD_BINARIESDIRECTORY}"/config
    cp doh-client/doh-client.conf "${BUILD_BINARIESDIRECTORY}"/config
}

# used to get version
function build_native() {
    echo "Building a native binary..."

    go build -ldflags="-s -w" -o ${BUILD_BINARIESDIRECTORY}/"${EXE}"-native
}

function build() {
    echo "Building ${EXE} for OS=$1 ARCH=$2"
    env GOOS="$1" GOARCH="$2" go build -ldflags="-s -w" -o ${BUILD_BINARIESDIRECTORY}/"${EXE}"-"$3"

    # echo "Compressing executable"
    # ! upx --ultra-brute ${BUILD_BINARIESDIRECTORY}/${EXE}-"$3" || true
}

function package() {
    VERSION=$("${BUILD_BINARIESDIRECTORY}/${EXE}-native" --version | head -n 1 | cut -d" " -f2)
    REVISION=$(git log --pretty=format:'%h' -n 1)

    echo "Packaging ${EXE} ${VERSION} for OS=$1 ARCH=$2 TYPE=$3 DST=$4"

    ! rm -rf "${TMP_DIRECTORY}"/*

    mkdir -p "${TMP_DIRECTORY}"/usr/bin
    cp "${BUILD_BINARIESDIRECTORY}"/"${EXE}"-"$3" "${TMP_DIRECTORY}"/usr/bin/"${EXE}"

    mkdir -p "${TMP_DIRECTORY}"/usr/lib/systemd/system
    cp "${BUILD_BINARIESDIRECTORY}"/systemd/"${EXE}".service "${TMP_DIRECTORY}"/usr/lib/systemd/system

    mkdir -p "${TMP_DIRECTORY}"/etc/dns-over-https
    cp "${BUILD_BINARIESDIRECTORY}"/config/"${EXE}".conf "${TMP_DIRECTORY}"/etc/dns-over-https

    mkdir -p "${TMP_DIRECTORY}"/etc/NetworkManager/dispatcher.d
    cp "${BUILD_BINARIESDIRECTORY}"/nm-dispatcher/"${EXE}" "${TMP_DIRECTORY}"/etc/NetworkManager/dispatcher.d

    # call fpm
    fpm --input-type dir \
        --output-type $4 \
        --chdir "${TMP_DIRECTORY}" \
        --package "${BUILD_ARTIFACTSTAGINGDIRECTORY}" \
        --name "${EXE}" \
        --description "${DESCR}" \
        --version "${VERSION}" \
        --iteration "${REVISION}" \
        --url "https://github.com/m13253/dns-over-https" \
        --vendor "Star Brilliant <coder@poorlab.com>" \
        --license "MIT License" \
        --category "net" \
        --maintainer "James Swineson <autopkg@public.swineson.me>" \
        --architecture "$2" \
        --force \
        .
}

cd "${DIR}"/..
prepare_env
make deps
build_common

pushd doh-server
export EXE="doh-server"
export DESCR="DNS-over-HTTPS Server"

build_native

build linux amd64 linux-amd64
package linux amd64 linux-amd64 deb
! package linux amd64 linux-amd64 rpm
package linux amd64 linux-amd64 pacman

build linux arm linux-armhf
package linux arm linux-armhf deb
! package linux arm linux-armhf rpm
package linux arm linux-armhf pacman

build linux arm64 linux-arm64
package linux arm64 linux-arm64 deb
! package linux arm64 linux-arm64 rpm
package linux arm64 linux-arm64 pacman
# build darwin amd64 darwin-amd64
# build windows 386 windows-x86.exe
# build windows amd64 windows-amd64.exe
popd

pushd doh-client
export EXE="doh-client"
export DESCR="DNS-over-HTTPS Client"

build_native

build linux amd64 linux-amd64
package linux amd64 linux-amd64 deb
! package linux amd64 linux-amd64 rpm
package linux amd64 linux-amd64 pacman

build linux arm linux-armhf
package linux arm linux-armhf deb
! package linux arm linux-armhf rpm
package linux arm linux-armhf pacman

build linux arm64 linux-arm64
package linux arm64 linux-arm64 deb
! package linux arm64 linux-arm64 rpm
package linux arm64 linux-arm64 pacman

# build darwin amd64 darwin-amd64
# build windows 386 windows-x86.exe
# build windows amd64 windows-amd64.exe
popd

