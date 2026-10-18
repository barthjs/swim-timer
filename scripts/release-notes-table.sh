#!/usr/bin/env bash

set -euo pipefail

# Prints the installation section of the release notes with the downloads, built from GoReleaser's dist/artifacts.json.
# GoReleaser cannot list the artifacts in the release notes on its own, so the release workflow appends this output
# to the changelog GoReleaser wrote. A kind of artifact the release does not contain leaves out its section.
# GitHub renders every line break of release notes as a break, so each paragraph of the output stays on one line.
#
# Usage: release-notes-table.sh <tag> <owner/repo> [artifacts.json]

TAG="${1:?tag required}"
REPO="${2:?owner/repo required}"
ARTIFACTS_JSON="${3:-dist/artifacts.json}"

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

# Sort position, operating system, and architecture per platform. Unknown platforms come last under their Go names.
PLATFORMS='{
  "linux/amd64":   [10, "Linux", "x86_64"],
  "linux/arm64":   [20, "Linux", "ARM64"],
  "linux/armv7":   [30, "Linux", "ARMv7"],
  "linux/armv6":   [31, "Linux", "ARMv6"],
  "linux/386":     [40, "Linux", "x86 (32-bit)"],
  "linux/riscv64": [50, "Linux", "RISC-V 64-bit"],
  "windows/amd64": [60, "Windows", "x86_64"],
  "windows/arm64": [61, "Windows", "ARM64"],
  "windows/386":   [62, "Windows", "x86 (32-bit)"]
}'

# count prints the number of artifacts of the given type.
count() {
  jq --arg type "$1" 'map(select(.type == $type)) | length' "$ARTIFACTS_JSON"
}

# rows prints one table row per artifact of the given type, ordered by platform.
# With "os" as the second argument, the row starts with the operating system, otherwise with the architecture.
rows() {
  jq -r --arg type "$1" --arg columns "${2:-}" --arg base "$BASE_URL" --argjson platforms "$PLATFORMS" '
    def platform: .goos + "/" + .goarch + (if (.goarm // "") != "" then "v" + .goarm else "" end);
    def entry: $platforms[platform] // [99, .goos, .goarch];
    map(select(.type == $type))
    | sort_by(entry[0])
    | .[]
    | (if $columns == "os" then "| \(entry[1]) " else "" end) + "| \(entry[2]) | [\(.name)](\($base)/\(.name)) |"
  ' "$ARTIFACTS_JSON"
}

# image prints the repository of the Docker image without its tag.
image() {
  jq -r 'map(select(.type == "Docker Image")) | first | .name | sub(":[^:/]*$"; "")' "$ARTIFACTS_JSON"
}

# image_platforms prints the labels of all platforms of the Docker image, ordered by platform and separated by commas.
image_platforms() {
  jq -r --argjson platforms "$PLATFORMS" '
    map(select(.type == "Docker Image") | .extra.Platforms[])
    | unique
    | map(sub("/v"; "v") as $platform | $platforms[$platform] // [99] + ($platform | split("/")))
    | sort_by(.[0])
    | map(.[1] + " " + .[2])
    | join(", ")
  ' "$ARTIFACTS_JSON"
}

echo "## Installation"

if [ "$(count "Archive")" -gt 0 ]; then
  cat <<MARKDOWN

### Archives

| OS | Architecture | Download |
|---|---|---|
$(rows "Archive" os)
MARKDOWN
fi

if [ "$(count "Linux Package")" -gt 0 ]; then
  cat <<MARKDOWN

### Debian and Ubuntu

\`\`\`sh
sudo apt install ./<package>.deb
\`\`\`

| Architecture | Package |
|---|---|
$(rows "Linux Package")
MARKDOWN
fi

if [ "$(count "Docker Image")" -gt 0 ]; then
  cat <<MARKDOWN

### Docker

Platforms: $(image_platforms)

\`\`\`sh
docker pull $(image):${TAG#v}
\`\`\`
MARKDOWN
fi

if [ "$(count "Checksum")" -gt 0 ]; then
  cat <<MARKDOWN

[checksums.txt](${BASE_URL}/checksums.txt) lists the SHA-256 checksum of every file.
MARKDOWN
fi
