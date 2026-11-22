---
title: Records
description: How Swim-Timer finds the records a heat broke or equalled.
sidebar:
  order: 4
---

Swim-Timer checks a heat with every time of a lane, so a record shows in the overlays as soon as its lane touches.
A corrected time is checked again. The records are only stored once the timing system marks the heat official,
and a heat that is made official again, for example after a correction, replaces the records it stored before.
[Records](../guides/records.md) in the guides describes how to import the record tables.

## Lanes that take part

A lane takes part if it has a finish time and a rank, and the start list names its competitor. Relays are not
checked, and neither are heats in a pool of another length than 25 m or 50 m, since a record belongs to one of them.

## Matching records

A record matches a lane if all of the following agree:

- the course, `SCM` for a 25 m pool and `LCM` for a 50 m pool, taken from the pool length of the heat,
- the category of the competitor, compared with the gender of the record,
- the distance of the event,
- the abbreviation of the style of the event, compared with the stroke of the record,
- the age range of the record, which has to cover the age of the competitor.

The age of a competitor is the year of the event minus the year of birth, as the competition age is counted.
Without a date of the event, the current year applies. A competitor without a year of birth only matches records
without an age range.

## Broken and equalled records

The time of a lane is compared in hundredths of a second. A faster time breaks the record, an equal time equals it.
Approved and pending records of a table count alike.

The enabled tables are checked in their order, and the first table in which the lane breaks or equals a record
decides. In that table, the lane counts for every age range whose record it breaks or equals. A table that lists
an age range twice counts the faster record.

A record set earlier at the meet takes the place of the record of its table while it is faster. The lanes of a heat
are checked from the fastest time on, so of two lanes under a record only the faster one sets it, and a lane with
the same time equals the new record. A heat that is checked again after a correction is only checked against
the records set before its own, and a record it sets again keeps the time at which it was first set.

## Disqualifications

The timing system does not transmit disqualifications, so a disqualified swimmer can set a record. Remove such
a record on the Records page. The next heats are then checked against the record that stood before it.
