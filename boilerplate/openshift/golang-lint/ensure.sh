#!/usr/bin/env bash
set -x
set -eo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source $REPO_ROOT/boilerplate/_lib/common.sh

GOLANGCI_LINT_VERSION="1.30.0"
OPM_VERSION="v1.15.2"
GRPCURL_VERSION="1.7.0"
DEPENDENCY=${1:-}
GOOS=$(go env GOOS)

case "${DEPENDENCY}" in

golangci-lint)
    GOPATH=$(go env GOPATH)
    if which golangci-lint ; then
        exit
    else
        mkdir -p "${GOPATH}/bin"
        echo "${PATH}" | grep -q "${GOPATH}/bin"
        IN_PATH=$?
        if [ $IN_PATH != 0 ]; then
            echo "${GOPATH}/bin not in $$PATH"
            exit 1
        fi
        DOWNLOAD_URL="https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-${GOOS}-amd64.tar.gz"
        curl -sfL "${DOWNLOAD_URL}" | tar -C "${GOPATH}/bin" -zx --strip-components=1 "golangci-lint-${GOLANGCI_LINT_VERSION}-${GOOS}-amd64/golangci-lint"
    fi
    ;;

*)
    echo "Unknown dependency: ${DEPENDENCY}"
    exit 1
    ;;
esac
