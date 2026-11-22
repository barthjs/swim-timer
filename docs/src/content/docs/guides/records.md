---
title: Records
description: Import record tables and announce the records broken at the meet in the overlays.
sidebar:
  order: 3
---

Swim-Timer checks every heat against the record tables. A broken or equalled record appears in the info overlay,
in the start list, and in the AR overlay. [Records](../concepts/records.md) describes how a record is found.

## Importing a record table

1. Open the Records page and choose New record table in the menu Record table.
2. Enter a name and an abbreviation. The abbreviation marks the records of the table on the Records page.
3. Choose the file. Its encoding and separator are detected.
4. Assign a column of the file to every column of a record. Columns with a known name are assigned already.
5. Translate the values of the file that the record tables or the meet data write differently.
   Common names are translated already.
6. Import the file.

The import reads CSV files of spreadsheets and other programs. If a row cannot be read, the dialog lists every
failed row with its line, and nothing is imported. [Record files](../reference/record-files.md) describes
the columns and their values.

Import replaces the records of an existing table with the records of a new file. Export downloads a table
in the format of the record files.

## Order of the tables

Only enabled tables are checked, in the order the choice of the record table lists them. A lane counts for the first
table in which it breaks or equals a record. The menu Record table changes the order and edits, imports into, exports,
and deletes the chosen table.

## Records set at the meet

The second tab lists every record set or equalled at the meet. The next heats are checked against these records,
so a slower time later in the meet does not count as a record as well.
The timing system does not transmit disqualifications, so a record of a disqualified swimmer has to be removed
by hand. Remove all records before the next meet.

## Files in the data directory

The record tables and the records set at the meet are stored in the folder `records` of the data directory
and held in memory. Reload reads them again after a file was changed by hand.
A CSV file copied into the folder appears as a disabled table. If a file cannot be read, Reload names the file
and the line, and the record tables cannot be changed until every file can be read again.
