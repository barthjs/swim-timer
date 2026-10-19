---
title: Updates
description: Check for a new release and install it.
sidebar:
  order: 6
---

Swim-Timer checks for a new release only when asked, with the update button next to the version at the bottom
of the web interface or with `swim-timer update check`. The button is shown after the login.

## Installing an update

An installation from the installation script for the current user, or from an unpacked release archive,
updates itself. The update downloads the release for the platform, verifies it against the published checksums,
replaces the binary, and restarts Swim-Timer. The web interface reloads once the new version answers.
A service keeps running across the restart.

`swim-timer update apply` installs the latest release from the command line. A running instance keeps the old version
until its next restart.

## Installations that update elsewhere

Some installations cannot replace their own binary. The check still reports a newer release, links to it,
and names the reason:

| Installation                    | How to update                                                                           |
| ------------------------------- | --------------------------------------------------------------------------------------- |
| Debian package                  | Install the package of the new release, see [Debian package](./debian.md#updates).      |
| Container                       | Pull the new image and recreate the container, see [Container](./container.md#updates). |
| System-wide installation script | Run the installation script again, then restart the service.                            |
| Binary in a read-only directory | Replace the binary by hand, or run the installation script again.                       |
| Build without a release version | Install a release.                                                                      |
