---
title: Special cases
description: What happens at a meet when something does not run as planned.
sidebar:
  order: 3
---

This page describes situations at a meet that need attention, and how Swim-Timer handles them.
The rules behind them are described in [Live heat](./live-heat.md).

## Heats that overlap

With many short heats, the next heat often starts before the previous one is ended at the timing system.
Its start signal then arrives late, and Swim-Timer corrects the running time with the
[start time correction](./start-time-correction.md):

1. The last swimmer of heat 3 touches, and the running time of heat 3 stops.
2. Heat 4 starts at 10:00:55. Heat 3 is not ended yet, so the timing system does not send this start.
3. Heat 3 is ended, and the start list overlay shows its results.
4. Heat 4 is loaded at 10:01:15, and the timing system now sends its start with the time of day 10:00:55.
5. The running time of heat 4 begins at 0:20 instead of 0:00.

If the previous heat is ended only after the next heat runs, the overlays stay on the running heat.
Ending the previous heat again later shows its results.

## Restarts

After a restart, Swim-Timer knows no heat until the timing system sends the next data, and the overlays keep
the last heat until then. Restart Swim-Timer before the first heat rather than during a session, since it measures
the clock difference to the timing system anew. A correction for a heat from before the restart shows only
the corrected time of that heat.

## Lane numbering

A pool numbered 0 to 9 whose lane 0 stays empty for the whole meet is counted from lane 1, since the start list
never uses lane 0. An AR preset for ten lanes is then shifted by one lane.

## Disqualifications

The timing system does not transmit disqualifications. A disqualified swimmer is ranked by the time like every
other swimmer. A lane in which nobody started is hidden as long as it has no time.

## Heats loaded very late

If a heat is loaded more than about 45 seconds after its start, the timing system numbers the first laps
inconsistently. A lane can show a wrong split or distance until the next lap. Finish times and ranks are not affected.

## Shared lanes on long distances

On long distances, two swimmers can share a lane. The timing system counts the touches of both as one lane,
so its times are meaningless, and the lane shows the swimmer listed last. Such heats are usually timed by hand
and shown without overlays.

## Relays

Relays are not supported yet. A relay lane shows the swimmer listed last in the start list, not the team.
