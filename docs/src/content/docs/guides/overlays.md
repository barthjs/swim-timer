---
title: Overlays
description: Add the live overlays to OBS or vMix.
sidebar:
  order: 3
---

The overlays are web pages with a transparent background that streaming software such as OBS or vMix
adds as browser sources. They fill the browser source at any size and update live while a heat runs.

| Overlay    | Content                                                                 |
| ---------- | ----------------------------------------------------------------------- |
| Timer      | The running time of the heat.                                           |
| Info       | The running time with the event and the heat.                           |
| Start list | The competitors of the heat, and the results once the heat is official. |
| Splits     | Every new lap or finish time for a few seconds.                         |
| Lane       | Name and club of the swimmer in one lane.                               |
| AR         | Name, rank, and time of each swimmer on the lanes of the camera image.  |

The dashboard lists every overlay with its URL. There is one lane overlay per lane of the live heat
and one AR overlay per [AR preset](./ar-presets.md).

## Adding an overlay

Copy the URL of the overlay from the dashboard and add it as a browser source in OBS or as a web browser input
in vMix, with the resolution of the stream as its size.

The streaming software has to reach Swim-Timer over the network. It usually rejects self-signed certificates,
so use plain HTTP or a trusted certificate for the overlays, see [HTTPS](./access.md#https).

## Behavior

The overlays stay empty until the Omega timing system reports the first heat. If an overlay loses the connection,
it keeps the last heat and reconnects on its own.

In the official results, the start list orders the swimmers by rank. [Live heat](../concepts/live-heat.md)
describes what the overlays show, and [Special cases](../concepts/special-cases.md) covers disqualifications.

## Browser compatibility

Streaming software renders browser sources with an embedded Chromium that is often several years old. The overlays
run in Chromium 90 and later, so they also work in the browser sources of older vMix versions.
