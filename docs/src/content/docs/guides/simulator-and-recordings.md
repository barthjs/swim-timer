---
title: Simulator and recordings
description: Record the data of the timing system and replay it without a timing system.
sidebar:
  order: 8
---

Swim-Timer records the data of the Omega timing system during a meet. Its simulator plays such a recording back
or generates heats of its own, so the overlays can be set up and checked without a timing system.

## Simulator

The simulator is set up in the Timing tab of the configuration page and requires Linux or Windows.

- With a simulated event and no recording, it plays the heats of that event from the loaded start list
  with random times in a loop.
- With a replayed recording, it plays the recorded data with the pauses as they were recorded, in a loop.
- With both, it plays only the heats of that event from the recording.

The replay uses the loaded meet data. Upload the meet files of the recorded meet to replay it with its names.

## Recording

Swim-Timer records the data of the timing system by default, one file per day in the folder `recordings`
of the storage. The recording only works with the local storage and never delays the overlays.

The Timing tab of the configuration page lists the recordings. A recording can be uploaded, chosen for the replay,
downloaded as it is or as a [table for analysis](../reference/recordings.md#table-for-analysis), and deleted.
