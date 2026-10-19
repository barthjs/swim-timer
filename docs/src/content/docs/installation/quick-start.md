---
title: Quick start
description: Install Swim-Timer with the installation script and open the web interface.
sidebar:
  order: 1
---

The installation script downloads the latest release for the computer and installs it for the current user.

On Linux:

```sh
curl -fsSLO https://raw.githubusercontent.com/barthjs/swim-timer/main/scripts/install.sh
bash install.sh --user
```

On Windows, in PowerShell 7:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/barthjs/swim-timer/main/scripts/install.ps1 -OutFile install.ps1
pwsh -File install.ps1 -Mode User
```

Without a service, start Swim-Timer from the application menu or the start menu. Then open `http://localhost:8080`
in a browser, or the address of the computer from other computers in the network.

Next, set a password for the login as described in [Configuration](../guides/configuration.md), and follow
the guides to [connect the timing system](../guides/timing-device.md) and add the [overlays](../guides/overlays.md).
