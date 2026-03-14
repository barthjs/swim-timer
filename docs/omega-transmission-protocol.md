# Omega Transmission Protocol

- Norm: RS 485
- Speed: 9600 to 115200 bds
- Data bits: 8
- Parity: No parity
- Stop bits: 1
- Other Code US ASCII / NO Handshake / No Xon-Xoff

Messages are transmitted in two consecutive parts.

- Part 1: `[SOH][STX][HOME]ABCDDEEFFFGG¬¬HH[EOT]`
- Part 2: `[SOH][STX][HOME][LF]JKK[STX]Hh:Mm:Ss.dc¬[EOT]`

## Part 1 (Bytes 1–20)

| Index | Byte  | Encoding | Field    | Description                                                                                                                                                                                                                                                                                                           |
|-------|-------|----------|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 0     | 1     | `0x01`   | `[SOH]`  | Start of transmission                                                                                                                                                                                                                                                                                                 |
| 1     | 2     | `0x02`   | `[STX]`  | Start of information                                                                                                                                                                                                                                                                                                  |
| 2     | 3     | `0x08`   | `[HOME]` | Control character                                                                                                                                                                                                                                                                                                     |
| 3     | 4     | ASCII    | `A`      | Message type:<br/><ul><li>`0` (`0x30`) = Ready at start</li><li>`1` (`0x31`) = Official End</li><li>`2` (`0x32`) = On Line Time</li><li>`3` (`0x33`) = Current race results</li><li>`4` (`0x34`) = Empty</li><li>`5` (`0x35`) = Previous race result</li></ul>                                                        |
| 4     | 5     | ASCII    | `B`      | Kind of time:<br/><ul><li>`S` (`0x53`) = Start</li><li>`I` (`0x49`) = Split time</li><li>`A` (`0x41`) = Finish</li><li>`D` (`0x44`) = Take over time in a relay</li><li>`R` (`0x52`) = Reaction time at start</li><li>`B` (`0x42`) = Button only at finish</li></ul>                                                  |
| 5     | 6     | ASCII    | `C`      | Time type:<br/><ul><li>`Space` (`0x20`) = Normal time</li><li>`E` (`0x45`) = Edited Time</li><li>`+` (`0x2B`) = Platform time after touchpad time</li><li>`-` (`0x2C`) = Platform time before touchpad time</li><li>`1` (`0x31`) = Button 1</li><li>`2` (`0x32`) = Button 2</li><li>`3` (`0x33`) = Button 3</li></ul> |
| 6-7   | 7-8   | Hex      | `DD`     | [Used lanes](#used-lanes) (bitmask)                                                                                                                                                                                                                                                                                   |
| 8-9   | 9-10  | ASCII    | `EE`     | Lap count                                                                                                                                                                                                                                                                                                             |
| 10-12 | 11-13 | ASCII    | `FFF`    | Event                                                                                                                                                                                                                                                                                                                 |
| 13-14 | 14-15 | ASCII    | `GG`     | Heat                                                                                                                                                                                                                                                                                                                  |
| 15-16 | 16-17 | `0x20`   | `¬¬`     | Space padding                                                                                                                                                                                                                                                                                                         |
| 17-18 | 18-19 | ASCII    | `HH`     | Rank                                                                                                                                                                                                                                                                                                                  |
| 19    | 20    | `0x04`   | `[EOT]`  | End of transmission                                                                                                                                                                                                                                                                                                   |

### Used lanes

The first byte is for the lanes 1 to 5, the second for the lanes 6 to 10. The least significant bits are
used one bit per lane. The bits 7,6 and 5 are fixed (0,0,1).

## Part 2 (Bytes 21–41)

| Index | Byte  | Encoding | Field    | Description           |
|-------|-------|----------|----------|-----------------------|
| 20    | 21    | `0x01`   | `[SOH]`  | Start of transmission |
| 21    | 22    | `0x02`   | `[STX]`  | Start of information  |
| 22    | 23    | `0x08`   | `[HOME]` | Control character     |
| 23    | 24    | `0x10`   | `[LF]`   | Start of Part 2       |
| 24    | 25    | ASCII    | `J`      | Lane number           |
| 25-26 | 26-27 | ASCII    | `KK`     | Current lap           |
| 27    | 28    | `0x02`   | `[STX]`  | Start of time field   |
| 28-38 | 29-39 | ASCII    | Time     | `Hh:Mm:Ss.dc`         |
| 39    | 40    | `0x20`   | `¬`      | `Space`               |
| 40    | 41    | `0x04`   | `[EOT]`  | End of transmission   |

## Control Characters

| Name | Hex    | Meaning                         |
|------|--------|---------------------------------|
| SOH  | `0x01` | Start of Header                 |
| STX  | `0x02` | Start of information            |
| HOME | `0x08` | Control character used by Omega |
| LF   | `0x10` | Start marker for Part 2         |
| EOT  | `0x04` | End of Transmission             |

## Status-Codes

| Code                 | Bedeutung | Beschreibung                                  |
|----------------------|:----------|:----------------------------------------------|
| `"0 "` (`0x30 0x20`) | Load      | A new run has been loaded                     |
| `"2S"` (`0x32 0x53`) | Start     | The start signal has been triggered           |
| `"2I"` (`0x32 0x49`) | Split     | Split time                                    |
| `"2A"` (`0x32 0x41`) | Finish    | Final time                                    |
| `"1 "` (`0x31 0x20`) | Official  | The race result has been officially confirmed |

## Time Format

`Hh:Mm:Ss.dc`

| Field | Description |
|-------|-------------|
| `Hh`  | Hours       |
| `Mm`  | Minutes     |
| `Ss`  | Seconds     |
| `dc`  | Deciseconds |
