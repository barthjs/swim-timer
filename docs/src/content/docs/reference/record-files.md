---
title: Record files
description: The columns and values of the record tables.
sidebar:
  order: 4
---

Swim-Timer stores every record table as a CSV file in the folder `records` of the data directory, encoded in UTF-8
with a byte order mark and separated by semicolons. Its header names the columns below by their keys. An export
writes the same format, so it can be imported again without assigning any column.

## Columns

| Key          | Content                                                                      | Required |
| ------------ | ---------------------------------------------------------------------------- | -------- |
| `course`     | `SCM` for a 25 m pool, `LCM` for a 50 m pool                                 | yes      |
| `gender`     | Abbreviation of the category, as in the meet files                           | yes      |
| `relayCount` | Number of swimmers per team, `1` for an individual record, the default       | no       |
| `distance`   | Distance each swimmer swims in meters, a relay leg for a relay               | yes      |
| `stroke`     | Abbreviation of the style, as in the meet files                              | yes      |
| `ageFrom`    | Lowest age the record applies to, empty or `0` for no lower bound            | no       |
| `ageTo`      | Highest age the record applies to, empty or `0` for no upper bound           | no       |
| `time`       | Record time as `[[h:]m:]s.hh`, with a point or a comma before the hundredths | yes      |
| `approved`   | `true` for an approved record, `false` for a pending one                     | no       |
| `name`       | Record holder                                                                | no       |
| `club`       | Club or nation of the record holder                                          | no       |
| `place`      | Place where the record was set                                               | no       |
| `date`       | Day the record was set, as `YYYY-MM-DD` or `D.M.YYYY` with optional zeros    | no       |
| `birthYear`  | Year of birth of the record holder                                           | no       |

The category and style abbreviations of the loaded meet data are offered as values in the import. A record
with another value is kept but matches no event.

## Importing other files

An imported file may have any columns and values. The import detects the encoding, UTF-8 or Windows-1252,
and the separator, a semicolon, a comma, or a tab. Columns are assigned by their key or a common English or German
name. The columns with fixed values, `course` and `approved`, need a translation for every value the file writes
differently, and the import only starts once each one has one. Common English and German names of the courses,
the styles, and yes and no are translated already.

## Files in the folder

The file `tables.json` keeps the name, the abbreviation, the state, and the order of every table. A CSV file
of the folder without an entry there is added as a disabled table named after the file. Such a file may leave out
the optional columns and may be encoded and separated like an imported file. Its header names the columns
by their keys or the common names the import assigns, since such a file is read without a mapping.
The file `breaks.json` keeps the records set at the meet. Reload on the Records page reads every file of the folder again.
