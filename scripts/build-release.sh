#!/bin/sh

set -e -f -u

if [ -z "$1" ]; then
    echo "Error: version argument required"
    echo "Usage: $0 <version>"
    exit 1
fi

APP_VERSION="$1"
APP_NAME="swim-timer"
BINARY_NAME="swim-timer"

export CGO_ENABLED=0

mkdir -p dist

LINUX_PLATFORMS="\
linux:386
linux:amd64
linux:arm64"
readonly LINUX_PLATFORMS

WINDOWS_PLATFORMS="\
windows:386
windows:amd64
windows:arm64"
readonly WINDOWS_PLATFORMS

build_linux() {
    build_os="$1"
    build_arch="$2"
    build_output="${BINARY_NAME}"

    echo "Building ${build_os} ${build_arch}..."
    GOOS="${build_os}" GOARCH="${build_arch}" VERSION="${APP_VERSION}" OUT="${build_output}" \
        sh ./scripts/go-build.sh

    tar -czf "dist/${APP_NAME}_${build_os}_${build_arch}.tar.gz" "${build_output}" LICENSE
    rm "${build_output}"
}

build_windows() {
    build_os="$1"
    build_arch="$2"
    build_output="${BINARY_NAME}.exe"

    echo "Building ${build_os} ${build_arch}..."
    GOOS="${build_os}" GOARCH="${build_arch}" VERSION="${APP_VERSION}" OUT="${build_output}" \
        sh ./scripts/go-build.sh

    zip -q "dist/${APP_NAME}_${build_os}_${build_arch}.zip" "${build_output}" LICENSE
    rm "${build_output}"
}

echo "Building frontend..."
export VITE_APP_VERSION="${APP_VERSION}"
cd web
npm ci --omit=dev
npm run build-only
cd ..

echo "$LINUX_PLATFORMS" | while read -r platform; do
    os=$(echo "$platform" | cut -d: -f1)
    arch=$(echo "$platform" | cut -d: -f2)
    build_linux "$os" "$arch"
done

echo "Generating Windows resources..."
go-winres make --in cmd/winres.json --out cmd/rsrc --product-version=git-tag --file-version=git-tag

echo "$WINDOWS_PLATFORMS" | while read -r platform; do
    os=$(echo "$platform" | cut -d: -f1)
    arch=$(echo "$platform" | cut -d: -f2)
    build_windows "$os" "$arch"
done

echo "Calculating checksums..."
(
    set +f
    cd dist
    : > checksums.txt
    for archive in ./*.zip ./*.tar.gz; do
        if [ -f "$archive" ]; then
            sha256sum "$archive" >> checksums.txt
        fi
    done
)

echo "Release build complete. Artifacts in dist/"
