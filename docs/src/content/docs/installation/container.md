---
title: Container
description: Run Swim-Timer as a container.
sidebar:
  order: 5
---

The container image `ghcr.io/barthjs/swim-timer` runs on `linux/amd64` and `linux/arm64`. Every release is tagged
with its full version, its minor and major version, and `latest`. Prereleases only get their full version.

## Compose file

```yaml
services:
  swim-timer:
    image: ghcr.io/barthjs/swim-timer:latest
    container_name: swim-timer
    restart: unless-stopped
    ports:
      - 8080:8080
    volumes:
      - ./data:/data
    env_file: .env
```

The `.env` file next to it holds the settings as environment variables, for example the time zone:

```sh
TZ=Europe/Berlin
```

[Settings](../reference/settings.md) lists every setting. Start the container with `docker compose up -d`.
The serial port of the timing system is passed into the container as described in
[Connecting the timing system](../guides/timing-device.md#container).

## Data volume

Everything Swim-Timer stores lies in the volume `/data`:

| Path                     | Content                                  |
| ------------------------ | ---------------------------------------- |
| `/data/config.yaml`      | The config file.                         |
| `/data/storage`          | The data of Swim-Timer.                  |
| `/data/logs`             | Log files, if enabled.                   |
| `/data/swim-timer/certs` | A generated TLS certificate, if enabled. |

The container runs Swim-Timer with the user and group ID 1000. `PUID` and `PGID` in the `.env` file set other IDs,
for example those of the owner of the `data` directory on the host.

## Updates

Pull the new image and recreate the container:

```sh
docker compose pull
docker compose up -d
```
