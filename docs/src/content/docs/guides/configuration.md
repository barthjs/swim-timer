---
title: Configuration
description: Change the settings of Swim-Timer.
sidebar:
  order: 6
---

Most installations need only a few settings: a password for the login and the connection to the timing system.
Everything else has a default that suits a typical meet.

## Changing settings

The configuration page of the web interface changes every setting. It saves them to the config file,
and they take effect after Swim-Timer restarts. Settings can also be set in the config file by hand,
or with environment variables and command-line flags, which take precedence over the file. A field set
by an environment variable or a flag shows a warning, since saving it in the file has no effect.

[Settings](../reference/settings.md) lists every setting with its environment variable and default.

## Common settings

| Setting     | Environment variable | Purpose                                                                            |
| ----------- | -------------------- | ---------------------------------------------------------------------------------- |
| Password    | `APP_SECRET`         | Protects the pages that need a login, see [Login](./access.md#login).              |
| Language    | `APP_LANGUAGE`       | Language of the web interface.                                                     |
| Serial port | `TIMING_SERIAL_PORT` | Port of the timing system, see [Connecting the timing system](./timing-device.md). |
| Baud rate   | `TIMING_BAUD_RATE`   | Has to match the timing system.                                                    |
| Pool length | `TIMING_POOL_LENGTH` | Length of the pool in meters.                                                      |
| Port        | `SERVER_PORT`        | Port of the web interface, 8080 by default.                                        |

Passwords and other secrets are never shown again after saving. Leaving their field empty keeps them.

## Colors

The Theme tab of the configuration page changes the colors of the web interface.
