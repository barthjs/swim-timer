---
title: Setup
description: Tools and commands to run Swim-Timer from source.
sidebar:
  order: 1
---

This page lists the tools to work on Swim-Timer and runs it with its web interface and documentation from source.
Contributions follow the [contributing guidelines](https://github.com/barthjs/swim-timer/blob/main/CONTRIBUTING.md).

## Requirements

| Tool                                         | Used for                                     |
| -------------------------------------------- | -------------------------------------------- |
| Go 1.27                                      | The backend.                                 |
| Node.js 24                                   | The web interface and the documentation.     |
| Docker                                       | The development server of the web interface. |
| [golangci-lint](https://golangci-lint.run) 2 | Formatting and linting of the Go code.       |
| [swag](https://github.com/swaggo/swag)       | The OpenAPI document of the API reference.   |
| [GoReleaser](https://goreleaser.com) 2       | Builds of the release artifacts.             |

## Repository

| Directory    | Content                                                                                    |
| ------------ | ------------------------------------------------------------------------------------------ |
| `cmd/`       | The entry point of the binary.                                                             |
| `internal/`  | The backend, see [Architecture](./architecture.md).                                        |
| `web/`       | The web interface, a Vue application built with Vite.                                      |
| `docs/`      | This documentation, an Astro site with Starlight, see [Documentation](./documentation.md). |
| `packaging/` | The systemd service and the scripts of the Debian package.                                 |
| `scripts/`   | The installation scripts, the entrypoint of the container, and release helpers.            |

## Running Swim-Timer

The backend embeds the built web interface and documentation, so build both once before the first start:

```sh
npm ci --prefix web
npm ci --prefix docs
npm run build-only --prefix web
npm run build-only --prefix docs
make run
```

`make run` starts two processes:

| Process               | Address                 | Content                                                                                                        |
| --------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------- |
| Backend with `go run` | `http://localhost:8080` | The API and the last build of the web interface and the docs.                                                  |
| Vite in a container   | `http://localhost:5173` | The web interface with changes applied at once, forwarding the API, the websocket, and `/docs` to the backend. |

A change to the backend needs a restart of `make run`.

Without a timing system, set a [simulated event](../guides/simulator-and-recordings.md#simulator) to get live heats.

The documentation has its own development server at `http://localhost:4321/docs/`:

```sh
npm run dev --prefix docs
```

## Building

| Command          | What it builds                                                                                                      |
| ---------------- | ------------------------------------------------------------------------------------------------------------------- |
| `make build`     | The binary for the current platform with GoReleaser into `bin/`, including the web interface and the documentation. |
| `make build-all` | The archives and packages of every platform, as a release does.                                                     |

Both install only the build tools in `web/` and `docs/` with `npm ci --omit=dev`.
Run `npm ci` there again afterward to get the development tools back.
