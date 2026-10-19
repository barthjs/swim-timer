---
title: AR presets
description: Set up the AR overlay, which shows the results on the lanes of the camera image.
sidebar:
  order: 4
---

The AR overlay shows the rank, the name, and the time of each swimmer directly on the lanes of the camera image,
following the perspective of the camera. A lane appears once its swimmer has finished.

An AR preset describes where the lanes appear in the image of one camera position. Each preset has its own
AR overlay, which the dashboard lists. Presets are created in the editor.

## Setting up a preset

1. Create a preset with the number of lanes the camera shows.
2. Set the resolution of the browser source and choose a snapshot of the camera image as background image.
   A snapshot from the streaming software shows the image exactly as the browser source covers it.
3. Drag the corners of lane 1 and of the last lane onto the pool in the image.
4. Align the lanes, which distributes the lanes in between following the perspective.
5. Correct single lanes if needed and lock them, so aligning again does not move them.
6. Set the lane order and the camera position, then save the preset.

The question marks and the help of the editor explain each setting and the shortcuts.

## Resolution

A preset stores the lanes relative to the image, so it fits every browser source with the same aspect ratio,
such as 1280 x 720 and 1920 x 1080. With another aspect ratio, the lanes are distorted.

## Lane order

The editor numbers the lanes of a preset from top to bottom in the camera image. The lane order selects whether
the top lane is lane 1 or the highest lane of the pool. Pools that count their lanes from 0 are recognized
from the start list, so a preset fits both numberings without a setting.

## Background image

The background image stays in the browser and is never uploaded. Another browser or computer therefore shows
the preset without it.

## Sharing presets

A preset can be downloaded as a file and imported in another installation, where it replaces a preset with the same
name once it is saved. Renaming a preset changes the URL of its AR overlay, so browser sources that show it
need the new URL. An open AR overlay picks up a saved preset when it is reloaded.
