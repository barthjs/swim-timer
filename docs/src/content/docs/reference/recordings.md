---
title: Recordings
description: Recording, replay, and format of the recordings of the timing data.
sidebar:
  order: 4
---

[Simulator and recordings](../guides/simulator-and-recordings.md) describes how to record and replay
the data of the timing system.

## Recording

With `TIMING_RECORDING` on, Swim-Timer records the data of the timing system in `recordings/` of the storage:

- Every day gets a file such as `recording_2026-10-18.jsonl`, named after the day Swim-Timer started. A restart
  on the same day appends to that file, and a running instance keeps its file past midnight.
- Only the `local` storage driver records, since a recording grows while it runs.
- A replay is not recorded. The heats the simulator generates are recorded and marked as such.
- If the disk cannot keep up, data is dropped, and the recording notes how many bytes are missing at that point.
- An upload replaces a recording of the same name, except the one being written.
- Neither the recording being written nor the replayed one can be deleted.

## Replay

The simulator sends its data through a virtual serial port, so it takes the same path as the data of a real
timing system. It sends a recording with the pauses between the reads as they were recorded, including the pause
of a restart. Every pass begins after a pause of 10 seconds with the first data, so the time between the start
and the first data is skipped. A clock that was set back between two lines counts as no pause. The clock of the
timing system jumps back with every pass and every restart in the recording, so the
[start time correction](../concepts/start-time-correction.md) measures its baseline anew at these points.

With a simulated event, the simulator replays only the heats of that event. A heat spans the data from the first byte
of its first frame to the last byte of its last frame, including noise and the frames of other events in between.
The next load frame ends it, so a later frame of the heat, such as a correction, is replayed with a short window
of its own. The pauses between the heats stay as they were recorded.

If the loaded meet data differs from the meet files in effect when the data was recorded, a warning is logged,
and the overlays may show wrong or missing names. With a simulated event, only the event, its start entries,
and their competitors have to match.

## Format

A recording is a [JSON Lines](https://jsonlines.org/) file. Every line is an object with a `type` and the time `at`
in ISO 8601 with the UTC offset of the server. There are four types of lines:

| Type      | Content                                                                                                                                                                                                            |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `session` | Start of Swim-Timer, with the format version, the version of Swim-Timer, the source, the port, and the baud rate.                                                                                                  |
| `meet`    | The meet files in use from this point on, each with its name, its SHA-256 hash, and its content in the original encoding, base64 encoded. Written at the start and after every successful reload of the meet data. |
| `data`    | The bytes of one read from the timing system as hex bytes, at the time they arrived.                                                                                                                               |
| `gap`     | The number of bytes dropped before this point.                                                                                                                                                                     |

The data lines hold the bytes as the timing system sent them, including noise and incomplete blocks. If Swim-Timer
stopped while writing, the last line of a session can be incomplete and is skipped when the recording is read.

## Table for analysis

`swim-timer recording csv <recording> [file]` turns a recording into a CSV table. It decodes the data as Swim-Timer
did while recording, with one row per session, meet line, frame, discarded block, and gap:

| Column                                             | Content                                                                                                                     |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `Time`                                             | Time of the row. A frame is complete with the read that brought its last byte.                                              |
| `Session`                                          | Number of the session in the recording, counted from 1.                                                                     |
| `Type`                                             | `session`, `meet`, `frame`, `discarded`, or `gap`.                                                                          |
| `Kind`                                             | Kind of a frame: `load`, `start`, `intermediate`, `finish`, `button`, `official`, or `unknown`.                             |
| `MsgType`, `MsgKind`, `TimeType`                   | The codes of the frame as the timing system sent them, see [Omega transmission protocol](./omega-transmission-protocol.md). |
| `Event`, `Heat`, `Lane`, `Lap`, `Rank`, `LapCount` | The values of the frame as the timing system counts them, heats from 1 and lane 10 as 0.                                    |
| `UsedLanes`                                        | The lanes in use, separated by spaces.                                                                                      |
| `TimeMS`                                           | The time of the frame in milliseconds, for a start frame the timing system's time of day.                                   |
| `ClockOffsetMS`, `BaselineMS`, `DelayMS`           | The values of the [start time correction](../concepts/start-time-correction.md) of a start frame.                           |
| `StartTime`                                        | The corrected start time of a start frame.                                                                                  |
| `DroppedBytes`                                     | The number of bytes of a discarded block or a gap.                                                                          |
| `Hex`                                              | The bytes of a frame or a discarded block.                                                                                  |
| `Details`                                          | The metadata of a session and the file names of a meet line.                                                                |

Every session starts with a new baseline of the start time correction, like a restart of Swim-Timer.
Noise in front of a frame appears as a discarded block at the time the following header arrived.
Bytes still pending at the end of a session appear as a discarded block as well.
