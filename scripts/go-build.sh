#!/bin/sh

set -e -f -u

GOOS="${GOOS:?GOOS not set}"
GOARCH="${GOARCH:?GOARCH not set}"
VERSION="${VERSION:-dev}"
OUT="${OUT:?OUT not set}"
CGO_ENABLED="${CGO_ENABLED:-0}"

export GOOS GOARCH CGO_ENABLED

LDFLAGS="-s -w -X 'main.Version=${VERSION}'"

if [ "$GOOS" = "windows" ]; then
    LDFLAGS="${LDFLAGS} -H windowsgui"
fi

mkdir -p "$(dirname "$OUT")"

echo "Building $GOOS/$GOARCH -> $OUT"
go build -ldflags="$LDFLAGS" -o "$OUT" ./cmd/
