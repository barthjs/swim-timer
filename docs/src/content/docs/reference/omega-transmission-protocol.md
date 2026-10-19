---
title: Omega transmission protocol
description: The frames the Omega timing device sends over its serial line.
sidebar:
  order: 6
---

- Norm: RS 485
- Speed: 9600 to 115200 bds, set on the device
- Data bits: 8
- Parity: No parity
- Stop bits: 1
- Other Code US ASCII / NO Handshake / No Xon-Xoff

Messages are transmitted in two consecutive parts. Numbers are right-aligned and padded with spaces,
except for the event number and the heat number, which have leading zeros.

- Part 1: `[SOH][STX][HOME]ABCDDEEFFFGG¬¬HH[EOT]`
- Part 2: `[SOH][STX][HOME][LF]JKK[STX]Hh:Mm:Ss.dc¬[EOT]`

## Part 1 (Bytes 1-20)

| Index | Byte  | Encoding | Field    | Description                                                                                                                                                                                                                                                                                                                                                                                                |
| ----- | ----- | -------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 0     | 1     | `0x01`   | `[SOH]`  | Start of transmission                                                                                                                                                                                                                                                                                                                                                                                      |
| 1     | 2     | `0x02`   | `[STX]`  | Start of information                                                                                                                                                                                                                                                                                                                                                                                       |
| 2     | 3     | `0x08`   | `[HOME]` | Control character                                                                                                                                                                                                                                                                                                                                                                                          |
| 3     | 4     | ASCII    | `A`      | Message type:<br/><ul><li>`0` (`0x30`) = Ready at start</li><li>`1` (`0x31`) = Official End</li><li>`2` (`0x32`) = On Line Time</li><li>`3` (`0x33`) = Current results</li><li>`4` (`0x34`) = Empty</li><li>`5` (`0x35`) = Previous result</li></ul>                                                                                                                                                       |
| 4     | 5     | ASCII    | `B`      | Kind of time:<br/><ul><li>`S` (`0x53`) = Start</li><li>`I` (`0x49`) = Split time</li><li>`A` (`0x41`) = Finish</li><li>`D` (`0x44`) = Take over time in a relay</li><li>`R` (`0x52`) = Reaction time at start</li><li>`B` (`0x42`) = Button only at finish</li></ul>                                                                                                                                       |
| 5     | 6     | ASCII    | `C`      | Time type:<br/><ul><li>`Space` (`0x20`) = Normal time</li><li>`E` (`0x45`) = Edited Time</li><li>`+` (`0x2B`) = Platform time after touchpad time, see [Further messages](#further-messages)</li><li>`-` (`0x2D`) = Platform time before touchpad time</li><li>`1` (`0x31`) = Number of the button 1</li><li>`2` (`0x32`) = Number of the button 2</li><li>`3` (`0x33`) = Number of the button 3</li></ul> |
| 6-7   | 7-8   | Bitmask  | `DD`     | [Used lanes](#used-lanes), `0x20` plus one bit per lane                                                                                                                                                                                                                                                                                                                                                    |
| 8-9   | 9-10  | ASCII    | `EE`     | Lap count: the total number of laps of the heat, the same in every frame of a heat (e.g. ` 4` for 200 m in a 50 m pool)                                                                                                                                                                                                                                                                                    |
| 10-12 | 11-13 | ASCII    | `FFF`    | Event number, as in the start list, with leading zeros (e.g. `003`)                                                                                                                                                                                                                                                                                                                                        |
| 13-14 | 14-15 | ASCII    | `GG`     | Heat number, counted from 1 (the start list counts heats from 0), with a leading zero (e.g. `02`)                                                                                                                                                                                                                                                                                                          |
| 15-16 | 16-17 | `0x20`   | `¬¬`     | Space padding                                                                                                                                                                                                                                                                                                                                                                                              |
| 17-18 | 18-19 | ASCII    | `HH`     | Rank of the swimmer at the lap of this time, see [Field values by message](#field-values-by-message)                                                                                                                                                                                                                                                                                                       |
| 19    | 20    | `0x04`   | `[EOT]`  | End of transmission                                                                                                                                                                                                                                                                                                                                                                                        |

### Used lanes

`DD` consists of two bytes. The first byte holds the lanes 1 to 5, the second the lanes 6 to 10.
The bits 7, 6, and 5 of each byte are fixed to `0`, `0`, and `1`, so an empty byte is `0x20`.
The five low bits mark one lane each. Bit 4 stands for the first lane of the byte (lane 1 or 6)
and bit 0 for the fifth (lane 5 or 10).

| Lanes in use | First byte          | Second byte         |
| ------------ | ------------------- | ------------------- |
| 3 to 5       | `0x27` (`00100111`) | `0x20` (`00100000`) |
| 1 to 8       | `0x3F` (`00111111`) | `0x3C` (`00111100`) |

The manufacturer's documentation shows the bits in reverse order. Transmissions follow the order described here.

## Part 2 (Bytes 21-41)

| Index | Byte  | Encoding | Field    | Description                                                       |
| ----- | ----- | -------- | -------- | ----------------------------------------------------------------- |
| 20    | 21    | `0x01`   | `[SOH]`  | Start of transmission                                             |
| 21    | 22    | `0x02`   | `[STX]`  | Start of information                                              |
| 22    | 23    | `0x08`   | `[HOME]` | Control character                                                 |
| 23    | 24    | `0x0A`   | `[LF]`   | Start of Part 2                                                   |
| 24    | 25    | ASCII    | `J`      | Lane number, a single digit, so a lane 10 can only arrive as `0`  |
| 25-26 | 26-27 | ASCII    | `KK`     | Lap: the number of the lap this time was taken at, counted from 1 |
| 27    | 28    | `0x02`   | `[STX]`  | Start of time field                                               |
| 28-38 | 29-39 | ASCII    | Time     | `Hh:Mm:Ss.dc`, see [Time format](#time-format)                    |
| 39    | 40    | `0x20`   | `¬`      | `Space`                                                           |
| 40    | 41    | `0x04`   | `[EOT]`  | End of transmission                                               |

## Control Characters

| Name | Hex    | Meaning                             |
| ---- | ------ | ----------------------------------- |
| SOH  | `0x01` | Start of Header                     |
| STX  | `0x02` | Start of information                |
| HOME | `0x08` | Control character used by Omega     |
| LF   | `0x0A` | Start marker for Part 2             |
| EOT  | `0x04` | End of Transmission                 |
| DC2  | `0x12` | Identification in the alive message |
| DC4  | `0x14` | Identification in the alive message |

The manufacturer's documentation lists LF as `10` hex. Devices send the line feed `0x0A`.

## Status-Codes

| Code                 | Name     | Description                               |
| -------------------- | -------- | ----------------------------------------- |
| `"0 "` (`0x30 0x20`) | Load     | A new heat has been loaded                |
| `"2S"` (`0x32 0x53`) | Start    | The start signal has been triggered       |
| `"2I"` (`0x32 0x49`) | Split    | Split time                                |
| `"2A"` (`0x32 0x41`) | Finish   | Final time                                |
| `"1 "` (`0x31 0x20`) | Official | The result of the heat has been confirmed |

## Field values by message

The fields of part 2 and the rank depend on the message:

| Message              | `HH` (rank) | `J` (lane) | `KK` (lap) | Time field                |
| -------------------- | ----------- | ---------- | ---------- | ------------------------- |
| Load, official       | spaces      | space      | spaces     | spaces                    |
| Start                | ` 0`        | `1`        | ` 0`       | Device's time of day      |
| Split, finish        | Rank        | Lane       | Lap        | Elapsed time of the heat  |
| Reaction time (`2R`) | `D `        | Lane       | spaces     | Reaction time with a sign |

Example of a heat with the lanes 1 to 8 in use, event 3, heat 2, and 4 laps:

```
Load:   [SOH][STX][HOME]0¬¬?<¬400302¬¬¬¬[EOT]  [SOH][STX][HOME][LF]¬¬¬[STX]¬¬¬¬¬¬¬¬¬¬¬¬[EOT]
Start:  [SOH][STX][HOME]2S¬?<¬400302¬¬¬0[EOT]  [SOH][STX][HOME][LF]1¬0[STX]14:17:55.26¬[EOT]
Split:  [SOH][STX][HOME]2I¬?<¬400302¬¬¬3[EOT]  [SOH][STX][HOME][LF]2¬1[STX]¬¬¬¬¬¬21.89¬[EOT]
Finish: [SOH][STX][HOME]2A¬?<¬400302¬¬¬2[EOT]  [SOH][STX][HOME][LF]4¬4[STX]¬¬¬¬1:22.07¬[EOT]
```

`¬` stands for a space, `?<` are the bytes `0x3F 0x3C` of the used lanes.

## Time format

`Hh:Mm:Ss.dc`

| Field | Description            |
| ----- | ---------------------- |
| `Hh`  | Hours                  |
| `Mm`  | Minutes                |
| `Ss`  | Seconds                |
| `d`   | Tenths of a second     |
| `c`   | Hundredths of a second |

Leading fields without a value are sent as spaces together with their separator,
for example `      41.43` for 41.43 seconds or `    1:22.07` for one minute and 22.07 seconds.
The first field with a value is right-aligned with a space, every following field has two digits.
In a start frame, the time field holds the device's time of day at the start instead of the elapsed time of the heat,
for example ` 7:52:33.60`.

Swim-Timer accepts a time only in this layout, with the hours from 0 to 23, minutes and seconds from 0 to 59,
and two digits for tenths and hundredths. Numeric fields consist of digits with optional leading spaces.
A field that deviates from this, such as a reaction time with its sign, counts as empty.

## Further messages

A device sends more than the messages listed above. This application ignores them:

- Reaction times: `2R` with the reaction time in the time field, for example `     + 0.24`, and `D` in the rank field.
- Relay takeovers: `D` as kind of time. Like reaction times, they carry the time type `+` or `-`,
  which compares the platform time, measured at the platform of the starting block, with the touchpad time.
  `+` means the platform time lies after the touchpad time, `-` that it lies before it, so in a relay the swimmer left
  the block before the previous swimmer touched the pad.
- Message types other than `0`, `1`, `2`, and `7`, such as `3` and `5` with results.

Button presses at the finish arrive with message type `7` and `B` as kind of time,
with the number of the button `1`, `2`, or `3` as time type. A frame such as `7B1` reports that button 1
was pressed at the finish. It carries lane, lap, and time like an intermediate time,
and [Live heat](../concepts/live-heat.md) describes how Swim-Timer records it.

Swiss Timing devices also use a general message format for other purposes,
`[SOH][DC2]«p»[DC3]«if»[DC4]«cd»[STX][DLE]«XXYY»String¬«INFO»[EOT]`. It does not apply to the swimming frames described
here. A device can also send an optional alive message, `[SOH][DC2]9[DC4]TP[EOT]`, which the application does not
evaluate.

A time corrected at the device is sent again for the same lane and lap. It often carries the time type `E`,
but can also arrive without it. A correction of the last lap arrives as `2I` with `E`, not as `2A`.
The repeated time replaces the earlier time of the lane like any other time.

The manufacturer states that further message types can appear in the future. A receiver checks the structure
of every message exactly and ignores what it does not know. Swim-Timer accepts a frame only if both parts
have the length given above and every fixed byte (control characters, line feed, and the fixed spaces) is in place.
Every part starts with `[SOH][STX][HOME]`, so after a disturbance Swim-Timer resumes reading
at the next such header. It discards bytes before a header, a part cut-off without its `[EOT]`,
and blocks longer than a part, and keeps the valid frame that follows.

## Transmission order

The device sends a frame when the operator triggers it, not when the event in the pool happens.
A heat loaded only after its start sends its start frame seconds or minutes late, with the original time of day
in the time field. [Start time correction](../concepts/start-time-correction.md) explains how the displayed time
compensates for that. An official frame can confirm an earlier heat while the next heat is already running.
