---
title: Releases
description: How a version tag turns into binaries, packages, and a container image.
sidebar:
  order: 5
---

A version tag such as `v1.2.3` starts the release workflow:

1. The tests run as for a pull request.
2. GoReleaser builds the binaries and the Debian packages into a draft of the GitHub release.
3. The container image for `linux/amd64` and `linux/arm64` goes to the GitHub Container Registry.
4. The release is published with the changelog and a table of the downloads.

| Artifact                         | Content                                                             |
| -------------------------------- | ------------------------------------------------------------------- |
| `swim-timer_<os>_<arch>` archive | The binary for Linux and Windows on every supported architecture.   |
| `.deb` package                   | The binary with the system service, man pages, and completions.     |
| `checksums.txt`                  | The checksums the update and the installation scripts verify.       |
| `ghcr.io/barthjs/swim-timer`     | The container image, see [Container](../installation/container.md). |

- Every build embeds the web interface and the documentation of its version, so an instance always serves
  the documentation that matches it.
- The build tools of the web interface and the documentation are listed under `dependencies` in their
  `package.json`, because the release build installs the packages with `npm ci --omit=dev`.
- The release attaches the built documentation as `swim-timer_docs.tar.gz`, which the public documentation
  is built from.
- The changelog groups the commits by their prefix, so commit messages follow
  [Conventional Commits](https://www.conventionalcommits.org).

## Public documentation

The docs workflow publishes the documentation to GitHub Pages after every release and after every change
of the documentation on `main`. A deployment replaces the whole site, so the workflow:

1. builds `main` into the folder `main/`,
2. unpacks the `swim-timer_docs.tar.gz` of every published release into a folder named after its tag,
3. writes `versions.json` with all versions, a start page that leads to the newest stable release, and a 404 page,
4. deploys the site.

A release without the archive, such as one from before the documentation, is left out. The workflow needs two settings
of the repository:

- GitHub Actions as the source of Pages.
- A deployment rule of the `github-pages` environment that allows the tags `v*`, since a release deploys from its tag.
