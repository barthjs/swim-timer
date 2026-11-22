---
title: Architecture
description: How the backend, the web interface, and the documentation are structured.
sidebar:
  order: 2
---

Swim-Timer is one Go binary. It reads the timing system, holds the meet data and the live heat in memory,
and serves the web interface, the overlays, and this documentation, which are built beforehand and embedded.

## Layers

Requests and commands pass through three layers, and a layer only calls the layer below it:

| Layer    | Package                        | Role                                                                |
| -------- | ------------------------------ | ------------------------------------------------------------------- |
| Input    | `internal/api`, `internal/cli` | Read and validate requests and commands, then call a service.       |
| Services | `internal/service`             | Hold the rules and call the stores and the infrastructure packages. |
| Stores   | `internal/store`               | Read and write the data.                                            |

The services cover auth, configuration, meet data, presets, recordings, the simulator, the timing, updates, and webhooks.

Domain types in `internal/domain` are only used inside the backend. The API reads request types and writes response
types, so its format does not change when the internal model does.

## Assembly

`internal/app` is the only place that wires the layers and the packages together. It offers building blocks, such as
the file storage and the stores, and `app.New` assembles the whole server from them. The server and the feature tests
start the application through `app.New`. A command that needs only some parts builds just those, so it does not open
the timing system or start the HTTP API.

The configuration in `internal/config` reads the settings from their sources. The services and the API read
no configuration themselves. They take their own options, which `internal/app` fills from the configuration.
Only the service and the handlers that manage the configuration itself use its types.

## Packages

Self-contained parts of the domain have packages of their own, since they import nothing from the API or the services:

| Package              | Role                                                                      |
| -------------------- | ------------------------------------------------------------------------- |
| `internal/omega`     | Reads the frames of the Omega transmission protocol from the byte stream. |
| `internal/serial`    | Opens serial ports and serial-over-IP bridges.                            |
| `internal/recording` | Writes and reads the recordings of the timing data.                       |

Infrastructure packages hold no business logic and import neither `config` nor the domain types:

| Package                | Role                                                             |
| ---------------------- | ---------------------------------------------------------------- |
| `internal/server`      | Runs the HTTP server with its middleware and security headers.   |
| `internal/media`       | Stores files locally or in S3.                                   |
| `internal/session`     | Keeps the sessions of the login in memory.                       |
| `internal/ratelimiter` | Limits requests in fixed windows.                                |
| `internal/health`      | Answers the liveness and readiness probes.                       |
| `internal/logging`     | Writes the log to the standard output, a rotating file, or both. |
| `internal/update`      | Replaces the binary with the latest release.                     |
| `internal/i18n`        | Translates the texts the backend produces.                       |
| `internal/webhook`     | Delivers signed messages to webhooks.                            |
| `internal/ws`          | Sends messages to the connected websocket clients.               |

## Conventions

Constructors and functions take their parameters in this order:

1. `ctx context.Context`
2. the logger
3. configuration values or an options struct of the package
4. infrastructure dependencies, such as the storage
5. services and stores
6. callbacks, such as a function that stops the application
7. functional options as the variadic last parameter

Services use the concrete stores. An interface only exists where there are several implementations, such as
the storage drivers, or where a test replaces something outside the process, such as the timing system.

## Web interface

The web interface in `web/` is a Vue application with Nuxt UI and Tailwind CSS:

| Folder                                 | Content                                                                       |
| -------------------------------------- | ----------------------------------------------------------------------------- |
| `src/api`                              | The client of the API with one repository per resource.                       |
| `src/pages`                            | One component per page, with the layouts in `src/layouts`.                    |
| `src/components`                       | Components by feature, with reusable blocks such as `table` and `form`.       |
| `src/composables`                      | Reusable logic of the components.                                             |
| `src/lib`                              | Functions without Vue, such as the targets of the documentation in `docs.ts`. |
| `src/stores`                           | Pinia stores for state that several pages share.                              |
| `src/locales`                          | The texts of the interface in every language.                                 |
| `src/overlay.ts`, `src/OverlayApp.vue` | The entry of the overlays, separate from the web interface.                   |

The overlays are a second entry of the same Vite project, without Nuxt UI and Tailwind CSS, since streaming software
renders them with an old Chromium, see [Browser compatibility](../guides/overlays.md#browser-compatibility). They are
built for Chromium 90 and only use browser features available there.

The live heat reaches the overlays, the scoreboard, and the web interface through a websocket. Everything else goes
through the JSON API under `/api`, which the [API reference](/api/) describes. The websocket also carries:

| Message   | Sent                                                      | Data                                                           |
| --------- | --------------------------------------------------------- | -------------------------------------------------------------- |
| `hello`   | on connect                                                | the version and a hash of the settings the web interface reads |
| `health`  | on connect and when the timing device connects or drops   | the state of every dependency, as `/readyz` names it           |
| `changed` | when the meet data, a preset, or the timing status change | the name of the resource and of the changed item               |

- A page reloads once `hello` names another version or other settings, as soon as a reload loses nothing
  and nobody reads the page, since it is hidden or its route sets `meta.unattended`. Until then it offers the reload,
  and the next navigation loads a new page. Unsaved changes that `useUnloadGuard` or `useDiscardConfirm` watch,
  and every other `useReloadHold`, keep it from reloading on its own.
- A page fetches a resource again after `changed` and after every reconnect, since it may have missed a change.
- An uncaught error shows a screen that reloads the page after 10 seconds, and after twice as long for every error
  that returns. A navigation that finds a module of an older build missing loads its target as a new page.
- A navigation without an answer from the server, or with a gateway error of a reverse proxy in front of it,
  shows the offline page of the service worker, which tries again every 10 seconds.

## Translations

The web interface and the backend are translated into every supported language:

| Texts                           | Files                                         | Test                             |
| ------------------------------- | --------------------------------------------- | -------------------------------- |
| Web interface                   | `web/src/locales/*.json`                      | `web/src/locales.test.ts`        |
| Backend, such as error messages | `internal/i18n/locales/*.json`                | `TestLocales` in `internal/i18n` |
| Documentation                   | see [Languages](./documentation.md#languages) |                                  |

- English is the default language.
- Every text exists in every language with the same keys, and the tests fail for a missing key.
- A new or changed text updates every language in the same change.
- German texts address the user with "du". Texts under shared keys, such as `errors.*` and `common.*`, stay generic.

## Documentation

The documentation in `docs/` is an Astro site with Starlight, built into `docs/dist` and embedded into the binary
like the web interface. [Documentation](./documentation.md) describes how it is written and built.
