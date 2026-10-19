---
title: Settings
description: Every setting with its environment variable and default, and where the configuration is read from.
sidebar:
  order: 1
---

Every setting can be changed on the configuration page, in the config file, with an environment variable,
or with a command-line flag. [Configuration](../guides/configuration.md) describes the common settings.

## Sources

Each setting is taken from the first source that provides it:

| Priority | Source               | Notes                                                                   |
| -------- | -------------------- | ----------------------------------------------------------------------- |
| 1        | Command-line flag    | Only when actually passed on the command line.                          |
| 2        | Environment variable | Also populated from a `.env` file in the working directory, if present. |
| 3        | Config file          | Optional. See below.                                                    |
| 4        | Built-in default     | Default values.                                                         |

Every setting has one name in three forms:

| Form                 | Rule                                         | Example              |
| -------------------- | -------------------------------------------- | -------------------- |
| Environment variable | Upper case, words joined by `_`              | `LOG_MAX_AGE_DAYS`   |
| Command-line flag    | Lower case, words joined by `-`, `--` prefix | `--log-max-age-days` |
| Config-file key      | Lower case, nested by section                | `log.max_age_days`   |

Every environment variable also has a `_FILE` variant that names a file holding the value, for example
`APP_SECRET_FILE` for secrets of a container. Setting both is an error. All settings are checked at startup,
and every invalid setting is reported.

## Config file

The config file is optional and may be `YAML`, `JSON`, or `TOML`. `--config` or `CONFIG_FILE` sets its path.
Without one, Swim-Timer uses the first `config.yaml` it finds in the working directory, in the user configuration
directory under `swim-timer/`, and in `/etc/swim-timer/`. The configuration page saves to the file that was loaded,
or creates `config.yaml` in the first writable of these directories. Saved settings take effect after a restart.

## App

| Environment variable | Default      | Description                                                           |
| -------------------- | ------------ | --------------------------------------------------------------------- |
| `APP_NAME`           | `Swim-Timer` | Application name shown in the interface, empty for the built-in name. |
| `APP_ENV`            | `production` | Environment (development, staging, production).                       |
| `APP_LANGUAGE`       | `en`         | Language of the user interface (en, de).                              |
| `APP_SECRET`         |              | Password of the login, empty allows the login without a password.     |

Secrets such as the password never appear in the logs or in the web interface.

## Log

| Environment variable | Default  | Description                                                                        |
| -------------------- | -------- | ---------------------------------------------------------------------------------- |
| `LOG_LEVEL`          | `warn`   | Log level (debug, info, warn, error).                                              |
| `LOG_CHANNEL`        | `stdout` | Log output channel (stdout, file, both).                                           |
| `LOG_PATH`           |          | Directory of the log files, empty for the log directory of the system or the user. |
| `LOG_FORMAT`         | `json`   | Log format (json, console).                                                        |
| `LOG_MAX_SIZE_MB`    | `100`    | Log file size in megabytes that triggers a rotation before the daily one.          |
| `LOG_MAX_AGE_DAYS`   | `30`     | Days to keep rotated log files, -1 keeps them forever.                             |
| `LOG_COMPRESS`       | `true`   | Compress rotated log files with gzip.                                              |

The log file `swim-timer.log` rotates every day and when it reaches its maximum size. Without `LOG_PATH`,
it is written to the first writable of these directories:

| OS      | System-wide                     | User                                                                           |
| ------- | ------------------------------- | ------------------------------------------------------------------------------ |
| Linux   | `/var/log/swim-timer`           | `$XDG_STATE_HOME/swim-timer/logs`, by default `~/.local/state/swim-timer/logs` |
| Windows | `%ProgramData%\swim-timer\logs` | `%LOCALAPPDATA%\swim-timer\logs`                                               |
| macOS   | `/Library/Logs/swim-timer`      | `~/Library/Logs/swim-timer`                                                    |

If no directory is writable, Swim-Timer logs to the standard output only. A systemd service writes the standard output
to the journal, and the container keeps its log files in `/data/logs`.

## Server

| Environment variable            | Default               | Description                                                                                          |
| ------------------------------- | --------------------- | ---------------------------------------------------------------------------------------------------- |
| `SERVER_HOST`                   | `*`                   | Listen host, * for all interfaces.                                                                   |
| `SERVER_PORT`                   | `8080`                | Listen port.                                                                                         |
| `SERVER_TRUSTED_PROXIES`        | `127.0.0.0/8,::1/128` | Comma-separated CIDRs or IPs of proxies whose forwarding headers are trusted.                        |
| `SERVER_CORS_ALLOWED_ORIGINS`   |                       | Comma-separated origins allowed to call the API from another domain, * for any, empty disables CORS. |
| `SERVER_CORS_ALLOW_CREDENTIALS` | `false`               | Allow cross-origin requests to send cookies, requires explicit origins.                              |
| `SERVER_SSL_ENABLED`            | `false`               | Serve over HTTPS, with a self-signed certificate if ssl_cert_file and ssl_key_file are empty.        |
| `SERVER_SSL_CERT_FILE`          |                       | PEM-encoded TLS certificate file, requires ssl_key_file.                                             |
| `SERVER_SSL_KEY_FILE`           |                       | PEM-encoded private key file of ssl_cert_file.                                                       |

With HTTPS enabled and no certificate files set, Swim-Timer generates a self-signed certificate for `localhost`
and its local network addresses and stores it in the `certs` folder of the first writable configuration directory.
Explicit certificate files are used as they are and must exist. [Login and network access](../guides/access.md)
explains HTTPS, the trusted proxies, and the CORS settings.

## Storage

| Environment variable        | Default | Description                                                               |
| --------------------------- | ------- | ------------------------------------------------------------------------- |
| `STORAGE_DRIVER`            | `local` | Storage driver (local, s3).                                               |
| `STORAGE_PATH`              |         | Storage root directory of the local driver, empty for the data directory. |
| `STORAGE_S3_ENDPOINT`       |         | S3 endpoint URL of the s3 driver.                                         |
| `STORAGE_S3_REGION`         |         | S3 region of the s3 driver.                                               |
| `STORAGE_S3_BUCKET`         |         | S3 bucket of the s3 driver.                                               |
| `STORAGE_S3_ACCESS_KEY`     |         | S3 access key of the s3 driver.                                           |
| `STORAGE_S3_SECRET_KEY`     |         | S3 secret key of the s3 driver.                                           |
| `STORAGE_S3_USE_PATH_STYLE` | `true`  | Use path-style bucket addressing.                                         |

The `s3` driver requires the endpoint, the bucket, the access key, and the secret key. Without `STORAGE_PATH`,
the `local` driver stores its files in the first writable of these directories:

| OS      | System-wide                               | User                                                                |
| ------- | ----------------------------------------- | ------------------------------------------------------------------- |
| Linux   | `/var/lib/swim-timer`                     | `$XDG_DATA_HOME/swim-timer`, by default `~/.local/share/swim-timer` |
| Windows | `%ProgramData%\swim-timer`                | `%LOCALAPPDATA%\swim-timer`                                         |
| macOS   | `/Library/Application Support/swim-timer` | `~/Library/Application Support/swim-timer`                          |

The storage contains the AR presets in `presets/`, the meet files in `meet/`, and the recordings in `recordings/`.

## Theme

| Environment variable | Default | Description                                          |
| -------------------- | ------- | ---------------------------------------------------- |
| `THEME_PRIMARY`      |         | Primary color (hex), empty for the built-in color.   |
| `THEME_SECONDARY`    |         | Secondary color (hex), empty for the built-in color. |
| `THEME_SUCCESS`      |         | Success color (hex), empty for the built-in color.   |
| `THEME_INFO`         |         | Info color (hex), empty for the built-in color.      |
| `THEME_WARNING`      |         | Warning color (hex), empty for the built-in color.   |
| `THEME_ERROR`        |         | Error color (hex), empty for the built-in color.     |
| `THEME_NEUTRAL`      |         | Neutral color (hex), empty for the built-in color.   |

The theme colors the web interface.

## Timing

| Environment variable    | Default | Description                                                                                                                                            |
| ----------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `TIMING_SERIAL_PORT`    |         | Serial port of the timing system, host:port for TCP, empty scans all ports.                                                                            |
| `TIMING_BAUD_RATE`      | `9600`  | Baud rate of the timing system (9600, 19200, 38400, 57600, 115200).                                                                                    |
| `TIMING_POOL_LENGTH`    | `50`    | Pool length in meters for the distance per lap, 0 derives it from the event distance and the lap count.                                                |
| `TIMING_SIMULATE_EVENT` | `0`     | Event number the simulator plays instead of reading the timing system, from the replayed recording or with random times, 0 disables it.                |
| `TIMING_REPLAY_PATH`    |         | Storage path of a recording the simulator replays instead of reading the timing system, such as recordings/recording_2026-10-18.jsonl, empty for none. |
| `TIMING_RECORDING`      | `true`  | Record the data of the timing system in recordings/ of the local storage.                                                                              |

The computer has to use the time zone of the timing system, see [Time zone](../guides/timing-device.md#time-zone).

## Webhook

| Environment variable       | Default | Description                                                                             |
| -------------------------- | ------- | --------------------------------------------------------------------------------------- |
| `WEBHOOK_ALLOWED_NETWORKS` |         | Comma-separated CIDRs or IPs of private or local networks webhooks may be delivered to. |

See [Webhooks](./webhooks.md#secret-and-address).
