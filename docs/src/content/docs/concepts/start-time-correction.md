---
title: Start time correction
description: How the running time stays correct when a heat is loaded after its start.
sidebar:
  order: 2
---

The running time counts from the start of the heat. The Omega timing system only sends the start signal once
the heat is loaded, which often happens after the swimmers have started, especially when heats follow each other
closely. Counting from the arrival of the signal would put the running time behind for the whole heat.

## How the delay is measured

The start signal carries the time of day of the start on the clock of the timing system. Swim-Timer compares it
with its own clock when the signal arrives:

```
offset = time of day of Swim-Timer - time of day of the timing system
```

The offset is the difference between both clocks plus the delay of the signal. A signal is never early,
so the smallest offset since Swim-Timer started belongs to a signal without delay and serves as the baseline:

```
delay      = offset - baseline
start time = arrival - delay
```

The first start sets the baseline, and a later start with a smaller offset replaces it. Once two further starts
match the baseline within half a second, it counts as confirmed. A start that would lower a confirmed baseline
by more than two seconds carries a wrong time of day. It starts at its arrival and leaves the baseline unchanged.

## Example

The clock of the timing system runs three minutes ahead. A start at 10:03:00 on the timing system arrives
at 10:00:00, so the offset of −180 s becomes the baseline. The next heat starts at 10:08:00 but is loaded late,
and its signal arrives at 10:05:06. Its offset is −174 s and its delay 6 s, so its running time begins at 6 s.

## Clocks

The clocks of the timing system and the computer do not have to match, but they must not be set during a meet,
since a changed clock looks like a delayed signal. After setting a clock, restart Swim-Timer. A change between
daylight saving time and standard time does not affect the correction. A start signal without a valid time of day
starts at its arrival.
