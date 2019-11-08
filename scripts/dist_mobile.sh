#!/bin/bash
set -e

# Get the version from the environment, or try to figure it out.
if [ -z $VERSION ]; then
	VERSION=$(awk -F\" '/Version =/ { print $2; exit }' < src/version/version.go)
fi
if [ -z "$VERSION" ]; then
    echo "Please specify a version."
    exit 1
fi
echo "==> Building version $VERSION..."



GIT="$(git rev-parse --abbrev-ref HEAD) $(git show --oneline -s)"
# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that.
cd "$DIR"

# Delete the old dir
echo "==> Removing old directory..."
rm -rf build/pkgmobile
mkdir -p build/pkgmobile

# Do a hermetic build inside a Docker container.
docker run --rm  \
    -u `id -u $USER` \
    -e "BUILD_TAGS=$BUILD_TAGS" \
    -v "$(pwd)":/workspace/go/src/github.com/mosaicnetworks/babble \
    -w /workspace/go/src/github.com/mosaicnetworks/babble \
    mosaicnetworks/mobile:0.0.1 ./scripts/dist_mobile_build.sh

# Add "babble" and $VERSION prefix to package name.
rm -rf ./build/distmobile
mkdir -p ./build/distmobile
for FILENAME in $(find ./build/pkgmobile -mindepth 1 -maxdepth 1 -type f); do
  FILENAME=$(basename "$FILENAME")
	cp "./build/pkgmobile/${FILENAME}" "./build/distmobile/babble_${VERSION}_${FILENAME}"
done

# Make the checksums.
pushd ./build/distmobile
shasum -a256 ./* > "./babble_${VERSION}_SHA256SUMS"
echo "$GIT" > ./git.version
ZIP="./babble_${VERSION}_android_library.zip"
zip "$ZIP" ./*  -x "$ZIP"
popd

# Done
echo
echo "==> Results:"
ls -hl ./build/distmobile

exit 0
