---
title: Linux
description: Install Swim-Timer on Linux with the installation script or from a release archive.
sidebar:
  order: 2
---

On Linux, Swim-Timer runs as a desktop application or as a systemd service. On Debian and Ubuntu,
the [Debian package](./debian.md) is an alternative.

## Installation script

The installation script downloads the latest release and installs it in one of two modes:

| Mode                     | Binary                      | Service                                        |
| ------------------------ | --------------------------- | ---------------------------------------------- |
| User-wide (`--user`)     | `~/.local/bin/swim-timer`   | A user service of the current user             |
| System-wide (`--system`) | `/usr/local/bin/swim-timer` | A system service that runs as the current user |

```sh
curl -fsSLO https://raw.githubusercontent.com/barthjs/swim-timer/main/scripts/install.sh
bash install.sh --user
```

Without an option, the script asks for the mode. It also asks whether to install a service and adds a launcher
to the application menu. A user service starts with every login of the user, a system service with the computer.

## Release archive

Every release has an archive per architecture, which holds the binary:

| Architecture  | Archive                                                                                                                           |
| ------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| x86_64        | [swim-timer_linux_amd64.tar.gz](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_linux_amd64.tar.gz)     |
| ARM64         | [swim-timer_linux_arm64.tar.gz](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_linux_arm64.tar.gz)     |
| ARMv7         | [swim-timer_linux_armv7.tar.gz](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_linux_armv7.tar.gz)     |
| ARMv6         | [swim-timer_linux_armv6.tar.gz](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_linux_armv6.tar.gz)     |
| x86 (32-bit)  | [swim-timer_linux_386.tar.gz](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_linux_386.tar.gz)         |
| RISC-V 64-bit | [swim-timer_linux_riscv64.tar.gz](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_linux_riscv64.tar.gz) |

Unpack it into a directory of your choice and start the binary:

```sh
tar -xzf swim-timer_linux_amd64.tar.gz
./swim-timer
```

## Files

Swim-Timer looks for its config file in the working directory, then in `~/.config/swim-timer/`, then in `/etc/swim-timer/`.
It stores its data in `/var/lib/swim-timer` if that directory is writable, otherwise in `~/.local/share/swim-timer`.
[Settings](../reference/settings.md#config-file) describes the order in detail.

## Updates

A user-wide installation and an unpacked archive update themselves from the web interface, see [Updates](./updates.md).
A system-wide installation is updated by running the installation script again and restarting the service.
