---
title: Live heat
description: How Swim-Timer turns the data of the timing system into the heat the overlays show.
sidebar:
  order: 1
---

The live heat is the heat the Omega timing system currently reports, combined with the names from the
[meet data](../guides/meet-data.md). Swim-Timer sends it to the overlays, the scoreboard, and the webhooks.

The timing system reports when a heat is loaded, started, and made official, and sends a time whenever a swimmer
completes a lap or finishes. A lap is one length of the pool.

## Heat on display

Any data for another heat switches the live heat to that heat, so a heat also appears if its first messages
were missed. Swim-Timer keeps the last eight heats, so a late correction for a previous heat shows it again
with all its times. Data for heat 0 is ignored, since that heat lists the competitors who withdrew.

The names of the competitors and of the event come from the meet data. A heat of an event that is missing
from the meet data shows its times without names until the meet data is reloaded.

## Official heats

An official message for a heat other than the one on display does not switch the display, since it usually confirms
the previous heat while the next one is already running. The heat becomes official among the last eight heats
and shows as official when it is recalled later.

## Start signals

A start signal for a heat that is not official clears all its times, for example after a false start.
A repeated start signal with the same time of day changes nothing, and a start signal for an official heat
is ignored, so its results stay.

After the next heat is loaded, the timing system sometimes repeats the start signal of the previous heat under
the new number. Swim-Timer holds such a signal back. If a new start signal follows, the heat starts with that one.
If the first time of the heat arrives instead, the heat starts with the time of day of the held back signal.

The start time of every heat is corrected for a late start signal, see
[Start time correction](./start-time-correction.md).

## Times and finish

Swim-Timer keeps the time of every lane at every lap. A lane shows the time of its highest lap and is finished
once the timing system sends a finish time for it.

- A corrected time replaces the time of its lane and lap, including a corrected finish time.
- A finish that was taken too early and demoted to a split withdraws the finish of the lane until the real touch
  arrives as a new finish time.
- A time from a backup button fills a lap without a touch time. It never replaces a touch time and never finishes
  a lane.
- A time for a lane without a start entry adds the lane without a competitor.

## Ranks

Swim-Timer computes the ranks from the times of every lap. Equal times share a rank, and the next rank is skipped,
as in 1, 1, 3. A lane shows its rank at its highest lap. The ranks of the timing system are not used,
since it only updates the rank of the lane it reports.

## Lanes in use

The timing system reports which lanes are in use and drops a lane when nobody starts in it. A lane without any time
that is not in use is hidden, even if the start list names a competitor for it. A lane with a time always stays,
and if the timing system reports no lane in use, no lane is hidden.

## End of the heat

The timing system sends nothing after the last touch. Once every lane in use has a finish time, the running time
stops at the slowest finish, and a withdrawn finish lets it continue. If a lane in use never finishes, the running time
stops when the heat becomes official.

## Distance and pool length

The distance of a lane is its highest lap multiplied by the pool length. With a pool length of 0, Swim-Timer derives
the pool length from the distance of the event and the number of laps, for example 25 m for 200 m in 8 laps,
and uses 50 m without these values.

## Lane numbering

The timing system sends the lane as a single digit, so lane 0 and lane 10 arrive the same. Swim-Timer takes it
as lane 0 if any start in the start list uses lane 0, and as lane 10 otherwise. The overlays follow the numbering
of the pool without a setting.
