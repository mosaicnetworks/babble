#!/bin/bash
set -e

# VERSION is used to name the build files. If a string is passed as a parameter
# to this script, it will be used as the VERSION. Otherwise, we use a descriptor
# of the git commit - "<branch>_<commit-hash>"
VERSION=${1:-}

if [ -z "$VERSION" ]; then
  VERSION="$(git rev-parse --abbrev-ref HEAD)_$(git rev-parse HEAD)"
fi

echo "==> Building version: $VERSION..."

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that.
cd "$DIR"

# Delete the old dir
echo "==> Removing old directory..."
rm -rf build/pkg
mkdir -p build/pkg

# Do a hermetic build inside a Docker container.
docker run --rm  \
    -u `id -u $USER` \
    -e "BUILD_TAGS=$BUILD_TAGS" \
    -v "$(pwd)":/go/src/github.com/mosaicnetworks/babble \
    -w /go/src/github.com/mosaicnetworks/babble \
    mosaicnetworks/glider:0.0.4dev ./scripts/dist_build.sh

# Add "babble" and $VERSION prefix to package name.
rm -rf ./build/dist
mkdir -p ./build/dist
for FILENAME in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type f); do
  FILENAME=$(basename "$FILENAME")
	cp "./build/pkg/${FILENAME}" "./build/dist/babble_${VERSION}_${FILENAME}"
done

# Make the checksums.
pushd ./build/dist
shasum -a256 ./* > "./babble_${VERSION}_SHA256SUMS"
popd

# Done
echo
echo "==> Results:"
ls -hl ./build/dist

exit 0
