#!/usr/bin/env bash
set -e

# this script is used to update vendored dependencies
#
# Usage:
# vendor.sh
#   revendor all dependencies
# vendor.sh github.com/docker/engine-api
#   revendor only the engine-api dependency
# vendor.sh github.com/docker/engine-api v0.3.3
#   vendor only engine-api at the specified tag/commit
# vendor.sh git github.com/docker/engine-api v0.3.3
#   is the same but specifies the VCS for cases where the VCS is something else than git
# vendor.sh git golang/x/sys eb2c74142fd19a79b3f237334c7384d5167b1b46 https://github.com/golang/sys.git
#   vendor only golang.org/x/sys downloading from the specified URL

cd "$(dirname "$BASH_SOURCE")/.."
source 'build/.vendor-helpers.sh'

case $# in
0)
    rm -rf vendor/*
    ;;
# If user passed arguments to the script
1)
    eval "$(grep -E "^clone [^ ]+ $1" "$0")"
    exit 0
    ;;
2)
    rm -rf "vendor/src/$1"
    clone git "$1" "$2"
    clean
    exit 0
    ;;
[34])
    rm -rf "vendor/src/$2"
    clone "$@"
    clean
    exit 0
    ;;
*)
    >&2 echo "error: unexpected parameters"
    exit 1
    ;;
esac

# the following lines are in sorted order, FYI
clone git github.com/gorilla/context aed02d124ae4a0e94fea4541c8effd05bf0c8296
clone git github.com/gorilla/mux 9fa818a44c2bf1396a17f9d5a3c0f6dd39d2ff8e
clone git github.com/gorilla/securecookie ff356348f74133a59d3e93aa24b5b4551b6fe90d
clone git github.com/gorilla/sessions 56ba4b0a11da87516629a57408a5f7e4c8ea7b0b
clean
