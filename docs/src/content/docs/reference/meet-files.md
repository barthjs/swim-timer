---
title: Meet files
description: The export files of the timing system and the columns Swim-Timer reads.
sidebar:
  order: 3
---

The meet files are the export files of the Omega timing system: text files with the extension `.TXT`, separated
by semicolons and encoded in Windows-1252. Swim-Timer stores them as uploaded. Some column names are French
or misleading, so the tables below describe what each column holds. Other columns are not read.

## LSTCAT (categories)

| Column     | Content                                                              |
| ---------- | -------------------------------------------------------------------- |
| `Category` | Display name, for example `Women`                                    |
| `AbrevCat` | Abbreviation, referenced by `abCat` in LSTRACE and LSTCONC, e.g. `W` |

## LSTLONG (distances)

| Column      | Content                                                              |
| ----------- | -------------------------------------------------------------------- |
| `idLength`  | Key, referenced by `idLen` in LSTRACE                                |
| `Longueur`  | Display name of the distance, for example `4x100 m`                  |
| `Mlongueur` | Total distance in meters, all relay legs together, for example `400` |
| `Relais`    | Number of swimmers per team, `1` for an individual event             |

## LSTSTYLE (styles)

| Column       | Content                                 |
| ------------ | --------------------------------------- |
| `idStyle`    | Key, referenced by `idStyle` in LSTRACE |
| `Style`      | Display name, for example `Freestyle`   |
| `StyleAbrév` | Abbreviation, for example `FR`          |

## LSTCONC (competitors)

| Column      | Content                                                  |
| ----------- | -------------------------------------------------------- |
| `id`        | Key of the competitor, referenced by `idBib` in LSTSTART |
| `bib`       | Start number, not used, since it can be empty            |
| `lastname`  | Last name                                                |
| `firstname` | First name                                               |
| `birthyear` | Year of birth                                            |
| `abNat`     | Club, or the nation at international meets               |
| `abCat`     | Category, see LSTCAT                                     |

## LSTRACE (events)

| Column    | Content                                                        |
| --------- | -------------------------------------------------------------- |
| `event`   | Event number, the number the timing system transmits           |
| `round`   | Phase of the event, for example heats or final, see LSTROUND   |
| `nbHeat`  | Number of heats                                                |
| `idLen`   | Distance, see LSTLONG                                          |
| `idStyle` | Style, see LSTSTYLE                                            |
| `abCat`   | Category, see LSTCAT                                           |
| `date`    | Scheduled date as year, month, and day, for example `26/05/01` |
| `time`    | Scheduled start time, `00:00` for an event without one         |

An event is a group of heats over one distance and style. The timing system numbers every phase of an event,
such as heats and final, as an event of its own. The overlays name an event by its distance, style, and category,
such as `400 m Medley Women`.

## LSTROUND (phases)

| Column       | Content                               |
| ------------ | ------------------------------------- |
| `idRound`    | Key, referenced by `round` in LSTRACE |
| `TITLE`      | Display name, for example `Final`     |
| `RoundAbrev` | Abbreviation, for example `FIN`       |

The file is the same for every meet. Phase 3 is the final.

## LSTSTART (start list)

| Column   | Content                                                                                                     |
| -------- | ----------------------------------------------------------------------------------------------------------- |
| `event`  | Event, see LSTRACE                                                                                          |
| `round`  | Phase, see LSTRACE                                                                                          |
| `heat`   | Heat number counted from 0, while the timing system counts heats from 1, `-1` for a competitor who withdrew |
| `lane`   | Lane                                                                                                        |
| `relais` | Relay position, `0` in every individual event of the exported data                                          |
| `idBib`  | Competitor, see `id` in LSTCONC, despite the name not the start number                                      |

A start entry only belongs to its event if it has the phase of the event. Two competitors in the same lane
of a heat are described in [Special cases](../concepts/special-cases.md#shared-lanes-on-long-distances).
