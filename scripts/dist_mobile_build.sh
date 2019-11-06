#!/bin/bash
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that.
cd "$DIR"

# Get the git commit
GIT_COMMIT="$(git rev-parse --short HEAD)"
GIT_DESCRIBE="$(git describe --tags --always)"
GIT_IMPORT="github.com/mosaicnetworks/babble/src/version"

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-"solaris darwin freebsd linux windows"}

export XDG_CACHE_HOME=/tmp/.cache.$$

# Get Go deps
echo "USER: `id -u $USER`"
mkdir -p glide_cache
glide --home "glide_cache" install
rm -rf glide_cache


# 

# Build!
echo "==> Building..."

$(which gomobile) bind -v -target=android -tags=mobile -o /workspace/go/src/github.com/mosaicnetworks/babble/build/pkgmobile/mobile.aar github.com/mosaicnetworks/babble/src/mobile 

exit 0
