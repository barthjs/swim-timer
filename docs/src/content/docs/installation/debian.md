---
title: Debian package
description: Install Swim-Timer as a system service on Debian and Ubuntu.
sidebar:
  order: 4
---

The Debian package installs Swim-Timer with a system service. Download the package for the architecture
of the computer from the [releases page](https://github.com/barthjs/swim-timer/releases/latest) and install it:

```sh
sudo apt install ./swim-timer_*.deb
```

The service runs as the system user `swim-timer`.

| Path                          | Content                 |
| ----------------------------- | ----------------------- |
| `/etc/swim-timer/config.yaml` | The config file.        |
| `/var/lib/swim-timer`         | The data of Swim-Timer. |
| `/var/log/swim-timer`         | Log files, if enabled.  |

## Configure and start

Create a config file with every setting at its default, adjust it, then enable and start the service:

```sh
sudo -u swim-timer swim-timer config init /etc/swim-timer/config.yaml
sudo editor /etc/swim-timer/config.yaml
sudo systemctl enable --now swim-timer
```

The settings can also be changed in the web interface. They take effect after a restart
with `sudo systemctl restart swim-timer`. The service writes its log to the journal:

```sh
journalctl -u swim-timer
```

## Updates

Download the package of the new release and install it the same way. The web interface reports a newer release
and links to it, but does not install it.
