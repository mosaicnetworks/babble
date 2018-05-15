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
GIT_IMPORT="github.com/mosaicnetworks/babble/version"

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-"solaris darwin freebsd linux windows"}

# Get Go deps
echo "USER: `id -u $USER`"
mkdir -p glide_cache
glide --home "glide_cache" install
rm -rf glide_cache

# Build!
echo "==> Building..."
"$(which gox)" \
		-os="${XC_OS}" \
		-arch="${XC_ARCH}" \
		-osarch="!darwin/arm !solaris/amd64 !freebsd/amd64" \
		-ldflags "-X ${GIT_IMPORT}.GitCommit='${GIT_COMMIT}' -X ${GIT_IMPORT}.GitDescribe='${GIT_DESCRIBE}'" \
		-output "build/pkg/{{.OS}}_{{.Arch}}/babble" \
		-tags="${BUILD_TAGS}" \
		github.com/mosaicnetworks/babble/cmd/babble

# Zip all the files.
echo "==> Packaging..."
for PLATFORM in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type d); do
		OSARCH=$(basename "${PLATFORM}")
		echo "--> ${OSARCH}"

		pushd "$PLATFORM" >/dev/null 2>&1
		zip "../${OSARCH}.zip" ./*
		popd >/dev/null 2>&1
done

exit 0
