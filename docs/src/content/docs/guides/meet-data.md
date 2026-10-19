---
title: Meet data
description: Load the events, the start list, and the competitors from the export files of the timing system.
sidebar:
  order: 2
---

The meet data holds the events of a meet, their heats, the start list, and the competitors. Swim-Timer reads it
from the export files of the Omega timing system and adds the names and clubs to the times of the live heat.
Without meet data, the overlays show the times without names.

## Loading the meet data

1. Export the meet files from the timing system. Swim-Timer reads the files `LSTCAT`, `LSTCONC`, `LSTLONG`,
   `LSTRACE`, `LSTROUND`, `LSTSTART`, and `LSTSTYLE`.
2. Upload them on the Meet data page. Each file is matched by its name.
3. Reload the meet data. Every open page and overlay picks up the new data.

Uploading alone does not reload, so several files can be replaced before they are read together. A missing file
leaves its part of the meet data empty. [Meet files](../reference/meet-files.md) describes the content of each file.

## Files that cannot be read

If a file cannot be read, the reload fails with the name of the file and the line of the problem,
and the previous meet data stays in use. A file cannot be read if:

- a column that Swim-Timer reads is missing,
- a row ends early, as in a file that is still being written,
- a key occurs twice,
- it is not encoded in Windows-1252, as the timing system writes it.

A file that was opened and saved again in an editor is often stored as UTF-8 and is then rejected, since names
with special characters would come out garbled. Upload the files as the timing system exported them.

## Changes during the meet

Finals and other events added during the meet get their start list only then. Export the files again, upload them,
and reload the meet data. The heat on display takes the new names with the next data of the timing system.

## Meet overview

The Meet page shows the loaded meet: the events with their heats, the starts of every competitor, and the clubs.
The tables can be searched and filtered, and a click on a club, a competitor, or an event filters by it.
The current state names the heat the timing system reports and estimates how far the meet runs ahead of
or behind its schedule.

## Meet data in other programs

Other programs can read the loaded meet data without a login, as a whole or in parts such as the events,
the start list, or a single heat. The [API reference](/api/) describes the requests. A [webhook](./webhooks.md)
sends the meet data after every reload instead.
