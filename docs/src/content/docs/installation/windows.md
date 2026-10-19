---
title: Windows
description: Install Swim-Timer on Windows with the installation script or from a release archive.
sidebar:
  order: 3
---

On Windows, Swim-Timer runs as a desktop application or as a Windows service.

## Installation script

The installation script requires PowerShell 7. It downloads the latest release, verifies it against the published
checksums, installs it, and creates a shortcut in the start menu:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/barthjs/swim-timer/main/scripts/install.ps1 -OutFile install.ps1
pwsh -File install.ps1 -Mode User
```

| Mode     | Binary                                      | Working directory          |
| -------- | ------------------------------------------- | -------------------------- |
| `User`   | `%LOCALAPPDATA%\Programs\swim-timer`        | `%APPDATA%\swim-timer`     |
| `System` | `%ProgramFiles%\swim-timer`, requires admin | `%ProgramData%\swim-timer` |

Without `-Mode`, the script asks for the mode.

## Windows service

The script offers to install a Windows service named `SwimTimer`, which starts with Windows and restarts Swim-Timer
if it stops. The service requires [NSSM](https://nssm.cc), which can be installed with `winget install nssm`.
Without NSSM, the script skips the service, and running it again after installing NSSM adds the service.

## Release archive

Every release has a ZIP archive per architecture, which holds `swim-timer.exe`:

| Architecture | Archive                                                                                                                     |
| ------------ | --------------------------------------------------------------------------------------------------------------------------- |
| x86_64       | [swim-timer_windows_amd64.zip](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_windows_amd64.zip) |
| ARM64        | [swim-timer_windows_arm64.zip](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_windows_arm64.zip) |
| x86 (32-bit) | [swim-timer_windows_386.zip](https://github.com/barthjs/swim-timer/releases/latest/download/swim-timer_windows_386.zip)     |

Unpack it into a directory of your choice and start `swim-timer.exe`, which opens the web interface in the browser.

## Files

Swim-Timer looks for its config file in the working directory first. It stores its data in `%ProgramData%\swim-timer`
if that directory is writable, otherwise in `%LOCALAPPDATA%\swim-timer`.
[Settings](../reference/settings.md#config-file) describes the order in detail.

## Updates

Swim-Timer updates itself from the web interface, see [Updates](./updates.md). If it may not write to its own file,
the web interface says so, and running the installation script again installs the new version.
