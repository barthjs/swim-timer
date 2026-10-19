---
title: Login and network access
description: Protect the web interface, serve it over HTTPS, and run it behind a reverse proxy.
sidebar:
  order: 9
---

Swim-Timer is meant for a local network. This page describes the login, HTTPS, and what to set when a reverse proxy
or another website sits in front of it.

## Login

| Pages                                      | Login                                                     |
| ------------------------------------------ | --------------------------------------------------------- |
| Meet data page, editor, configuration page | Required                                                  |
| Dashboard, Meet page, scoreboard, overlays | Not required, so streaming software can load the overlays |

- The password is the setting `APP_SECRET`. Without it, the login accepts an empty password.
- Set a password whenever Swim-Timer is reachable from a network you do not control.

## HTTPS

- With HTTPS enabled and no certificate files set, Swim-Timer generates a self-signed certificate for its addresses
  in the local network, which browsers warn about.
- Streaming software usually rejects a self-signed certificate in browser sources, so use plain HTTP or a trusted
  certificate for the overlays.
- [Server](../reference/settings.md#server) describes the settings.

## Reverse proxy

- Swim-Timer takes the address of the client from the forwarding headers only if the proxy is trusted. By default,
  only proxies on the same computer are trusted.
- A proxy on another host or in a container network must be listed in `SERVER_TRUSTED_PROXIES`. Otherwise all clients
  share its address and its rate limits.
- The proxy has to forward websocket connections, since the overlays receive the live heat through one.
- The proxy should pass the security headers of Swim-Timer through.
- A proxy that serves only HTTPS may add `Strict-Transport-Security`.

## Websites that use the API

- A website on another domain can only use the API in the browser if its origin is listed in
  `SERVER_CORS_ALLOWED_ORIGINS`.
- `*` lets every origin read data, while requests that change data need the origin listed explicitly.
- Programs outside the browser, such as curl, are not affected.
