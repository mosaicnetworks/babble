#!/bin/bash
set -e

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that.
cd "$DIR"

export XDG_CACHE_HOME=/tmp/.cache.$$

# Build!
echo "==> Building..."

$(which gomobile) bind -v -target=android -tags="mobile" -o /workspace/go/src/github.com/mosaicnetworks/babble/build/pkgmobile/mobile.aar github.com/mosaicnetworks/babble/src/mobile 

exit 0
