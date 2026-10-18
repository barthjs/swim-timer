# Timing

Swim-Timer reads the results of each heat from an Omega (Swiss Timing) timing device over a serial line.
It combines them with the start list exported from the timing software and publishes the live heat
to every connected overlay.

## Device connection

The device sends its data as described in the [Omega transmission protocol](./omega-transmission-protocol.md),
without handshake, at the baud rate set on the device. `TIMING_BAUD_RATE` has to match it and defaults to 9600 baud.

With `TIMING_SERIAL_PORT` set, only that port is used. A value in `host:port` form connects to a serial-over-IP bridge,
such as ser2net or a Moxa device server, instead of a local port. Without a configured port, the system is scanned
for serial ports and every port found is tried in order. The first port that can be opened is used,
so on a machine with several serial ports, configure the right one explicitly.

A dropped connection is reopened every 5 seconds. If none of the known ports can be opened anymore,
for example because a USB adapter was plugged into another socket and got a new name, the system is scanned again.

In Docker, pass the device into the container (see `compose.yaml`). On Linux, only members of the `dialout` group
may open serial ports.

## Meet data

The meet data covers everything that happens at a meet: its events, their heats, the start list, and the competitors.
It comes from the files the timing software exports.

### Upload and reload

Upload the files on the Meet data page, then reload the meet data to read them. Uploading does not reload on its own,
so several files can be replaced before they are read together. Uploading and reloading require the admin login.

After a successful reload, every open page is notified and fetches the new data. The live heat takes the competitors
and the details of its event from the new data with the next frame of the device.
A file that does not exist leaves its part of the meet data empty.

### Files that cannot be read

If a file cannot be read, the reload fails with the name of the file and the line of the problem,
and the previous meet data stays in use. A file cannot be read if:

- its header lacks a column described below,
- a row ends early, as in a file that is still being written,
- a quoted value is not closed on its line, since it would otherwise take in the following rows,
- a key occurs twice,
- it was saved in another encoding than Windows-1252.

A file with a byte order mark is rejected, and so is a file that is valid UTF-8 as a whole and contains
special characters, such as umlauts, since an editor saved it again as UTF-8. Windows-1252 stores a special character
as a single byte, so an export with special characters is practically never valid UTF-8 as a whole. A file without
special characters reads the same in both encodings and is accepted. Rows without any value are skipped.

### Reading the meet data

Reading the loaded meet data requires no login, so pages for the control room and programs of other developers
can show the schedule and start lists. Heats are counted from 1 there, as the device counts them,
and dates are given as year, month, and day. The events are listed in the order of the program,
which is their order in LSTRACE within each day. Start entries of competitors who withdrew are left out.

Programs can read the whole meet at once or only a part of it: the events, the competitors ordered by name, the start
list ordered by event, heat, and lane, a single event, or one heat of an event. A heat lists its start entries by lane
and relay position, each with its competitor. Every answer carries the time the meet data was last loaded, so a program
that reads several parts can tell whether a reload happened in between. The API documentation describes the requests.

### File format

The files are semicolon-separated and Windows-1252 encoded, as the timing software writes them,
and are stored byte for byte. Files with the extension `.TXT`, as the timing software exports them, or `.txt`
are recognized. An uploaded file is stored as `.TXT`, and a copy with the lowercase extension is removed,
so the uploaded file is the one that is read.

The column names come from the timing software. Some are French, and some are misleading, so the tables below describe
what each column holds. Columns not listed are not read.

#### LSTCAT (categories)

| Column     | Content                                                              |
|------------|----------------------------------------------------------------------|
| `Category` | Display name, for example `Women`                                    |
| `AbrevCat` | Abbreviation, referenced by `abCat` in LSTRACE and LSTCONC, e.g. `W` |

#### LSTLONG (distances)

| Column      | Content                                                              |
|-------------|----------------------------------------------------------------------|
| `idLength`  | Key, referenced by `idLen` in LSTRACE                                |
| `Longueur`  | Display name of the distance, for example `4x100 m`                  |
| `Mlongueur` | Total distance in meters, all relay legs together, for example `400` |
| `Relais`    | Number of swimmers per team, `1` for an individual event             |

#### LSTSTYLE (styles)

| Column       | Content                                 |
|--------------|-----------------------------------------|
| `idStyle`    | Key, referenced by `idStyle` in LSTRACE |
| `Style`      | Display name, for example `Freestyle`   |
| `StyleAbrév` | Abbreviation, for example `FR`          |

#### LSTCONC (competitors)

| Column      | Content                                                  |
|-------------|----------------------------------------------------------|
| `id`        | Key of the competitor, referenced by `idBib` in LSTSTART |
| `bib`       | Start number, not used, since it can be empty            |
| `lastname`  | Last name                                                |
| `firstname` | First name                                               |
| `birthyear` | Year of birth                                            |
| `abNat`     | Club, or the nation at international meets               |
| `abCat`     | Category, see LSTCAT                                     |

#### LSTRACE (events)

| Column    | Content                                                        |
|-----------|----------------------------------------------------------------|
| `event`   | Event number, the number the device transmits                  |
| `round`   | Phase of the event, for example heats or final, see LSTROUND   |
| `nbHeat`  | Number of heats                                                |
| `idLen`   | Distance, see LSTLONG                                          |
| `idStyle` | Style, see LSTSTYLE                                            |
| `abCat`   | Category, see LSTCAT                                           |
| `date`    | Scheduled date as year, month, and day, for example `26/05/01` |
| `time`    | Scheduled start time, `00:00` for an event without one         |

An event is a group of heats over one distance and style, for example 200 m breaststroke male.
The timing software numbers every phase of an event, such as heats and final, as its own event.
The name shown in the overlays combines distance, style, and category, for example `400 m Medley Women`.
Events added during the meet, such as finals, have no scheduled start time.

#### LSTROUND (phases)

| Column       | Content                               |
|--------------|---------------------------------------|
| `idRound`    | Key, referenced by `round` in LSTRACE |
| `TITLE`      | Display name, for example `Final`     |
| `RoundAbrev` | Abbreviation, for example `FIN`       |

The file is a fixed catalog of the timing software and the same for every meet. Phase 3 is the final.
The frames of the device carry no phase, so the phase of the live heat comes from its event.

#### LSTSTART (start list)

| Column   | Content                                                                                              |
|----------|------------------------------------------------------------------------------------------------------|
| `event`  | Event, see LSTRACE                                                                                   |
| `round`  | Phase, see LSTRACE                                                                                   |
| `heat`   | Heat number counted from 0, while the device counts heats from 1, `-1` for a competitor who withdrew |
| `lane`   | Lane                                                                                                 |
| `relais` | Relay position, `0` in every individual event of the exported data                                   |
| `idBib`  | Competitor, see `id` in LSTCONC, despite the name not the start number                               |

A start entry only belongs to its event if it has the phase of the event. An entry of another phase
appears neither in the live heat nor in the meet overview.

On long distances, two competitors can share a lane of the same heat. Such heats are timed by hand,
and the overlays are usually not shown for them. The device may keep running and count the touches of both swimmers
as the laps of one lane, so its times of such a heat are meaningless. The lane shows the competitor listed last.

## Live heat

The live heat is the heat the timing device currently reports. The device announces a heat when it is loaded, started,
and made official, and sends a time whenever a swimmer completes a lap or finishes. A lap is one length of the pool.

### Heat on display

Every frame for another event or heat replaces the live heat with that heat, so a heat also appears if its load
and start frames were missed, for example after a restart of the application. The last eight heats keep their times,
so a late correction for the previous heat shows it again with its times instead of starting it empty.
The device counts heats from 1, and a frame for heat 0 is ignored,
since that heat would list the competitors who withdrew.

The competitors and the name of the event come from the meet data with every frame. A heat of an event
that is not part of the meet data is shown with its times but without names, and gets them with the next frame
after the meet data containing it is reloaded. If a reload removes the event of a heat that is already shown,
the heat keeps the names of its competitors and of its event.

### Official heats

An official frame for a heat other than the one on display does not switch the display. It confirms an earlier result
late, often while the next heat is already running, and switching to it would interrupt the running heat.
If the heat is among the last eight heats, it becomes official there, takes the lanes in use from the frame,
and stops its timers as described below, so recalling it later, for example through a correction,
shows it as official. A heat that is no longer among the last eight heats counts as unknown, and a time for it
shows that heat as running with only this time.

### Start signals

A start signal for a heat that is not official clears the times of all its lanes, for example on a restart
after a false start or after test touches. A start signal with the same time of day as the last start of its heat
repeats that start and changes nothing. A start signal for a heat that is official among the last eight heats
is ignored, so its results stay. A heat that has left the last eight heats counts as unknown,
and a start signal for it shows that heat as running.

After loading the next heat, the device sometimes repeats the start signal of the previous heat with the same time
of day under the new heat number. Either the repetition is stale and a new start signal follows, or the swimmers
did start under the old number and their times arrive under the new one. A start signal with the same time of day
as the last start of another heat is therefore held back, and the heat does not start yet. The first time for the heat
confirms it and starts the heat with the time of day of the held back signal. A start signal with another time of day
replaces the held back one.

The start time of a heat is corrected for the delay of its start signal,
see [Start time correction](#start-time-correction).

### Times and finish

The application keeps the time of every lane at every lap. A lane shows the time of its highest lap
and is finished once the device sends a finish time for it.

- Correction: the device sends a corrected time again for the same lane and lap, with or without the correction mark.
  It replaces only the time of that lap. A corrected finish therefore arrives as a repeated finish time,
  and a late correction of an earlier lap never replaces a finish on display.
- Early touch: if a touch at the last lap was taken as finish too early, the operator demotes it to a split,
  and the device sends an intermediate time for the last lap. It withdraws the finish of the lane: the lap keeps
  its time as a split, the lane is no longer finished, and it has no rank until the real touch arrives
  as a new finish time.
- Backup button: a button at the finish backs up the touchpad. Its time arrives as a button time for a lane and lap
  and is recorded as a split only if the lane has no time for that lap yet. It never replaces another time
  and never finishes a lane, also not at the last lap. A touch time that arrives later for the same lap replaces it.
- Lane without start entry: a time for such a lane adds it without a competitor, since someone swims in it.

### Ranks

The ranks are computed from the times of every lap. Equal times share a rank and the next rank is skipped,
as in 1, 1, 3. A lane shows its rank at its highest lap. An intermediate time at the last lap takes no part
in the ranks of that lap. The device sends a rank as well, but only updates it for the lane of the frame,
so a time entered late, for example from a backup button, would leave the ranks of the other lanes outdated.

### Lanes in use

Every frame carries the lanes in use as set on the device. The device drops a lane from them when nobody started in it,
usually while the heat runs.

A lane without any time that is not among the current lanes in use of its heat is left out of the live heat,
even if the start list names a competitor for it. It appears again with its competitor once the device reports it
in use again. A lane with a time always stays, since dropping a lane is no reliable sign of a disqualification.
If the device reports no lane in use, no lane is hidden. A frame whose field of the lanes in use is invalid
keeps the lanes in use of the frame before.

### End of the heat

The device sends nothing after the last touch, and the heat can take minutes to become official.
Once every lane in use has a finish time, the timers stop at the highest of these times. A withdrawn finish
lets them run again. If a lane in use never finishes, for example because it stays empty,
the timers stop when the heat becomes official.

### Distance and pool length

The distance of a lane is its highest lap multiplied by the pool length. With `TIMING_POOL_LENGTH` set to 0,
the pool length is derived from the distance of the event and the number of laps the device transmits,
for example 200 m in 8 laps for a 25 m pool. If the event is not part of the meet data, has no distance,
or the device transmits a lap count of 0, the pool length is 50 m. A frame with an empty or invalid lap count is
discarded.

### Lane numbering

The lane field of a frame has room for a single digit. A frame for lane `0` is assigned to lane 0
if any start entry of the meet uses lane 0, and otherwise to lane 10. The live heat reports the first lane of the pool,
0 or 1, so the overlays map their lanes without a setting.

## Start time correction

The running time in the overlays counts from the start time of the heat. The device only sends the start signal
once the heat is loaded, and heats are often loaded after they have already started,
especially when heats follow each other closely. The delay can range from a few seconds to several minutes.
Counting from the arrival would put the displayed time behind by that delay for the whole heat.

A start frame carries the time of day of the start on the device's clock. The application compares it with its own time
of day when the frame arrives:

```
offset = local time of day - device time of day
```

The offset consists of the difference between both clocks plus the transmission delay. A start signal is only
ever delayed, never early, so the smallest offset of any start since the application started is the offset
of a signal that arrived without delay. It is the baseline:

```
delay      = offset - baseline
start time = arrival - delay
```

The first start sets the baseline. A later start with a smaller offset replaces it, which corrects a first start
that arrived late itself.

Once two further starts arrived within half a second of the baseline, it counts as confirmed. The clocks drift
by milliseconds per hour, so a start that would lower a confirmed baseline by more than two seconds carries
a wrong time of day or comes after a clock was set. Such a start begins at its arrival and leaves the baseline
unchanged. Late starts are not limited by this, however long their delay.

For example, the device clock runs three minutes ahead of the host. A start at 10:03:00.00 on the device arrives
at 10:00:00.00 local time, so the offset is −180 s and becomes the baseline. The next heat starts at 10:08:00.00
on the device but is loaded late, and its start frame arrives at 10:05:06.00. Its offset is −174 s,
so the delay is 6 s and the displayed time begins at 6 seconds instead of zero.

The offsets are wrapped into a range of ±12 hours, so a heat that starts shortly before midnight
and arrives after it keeps a continuous offset.

The application reads its own time of day with the UTC offset the server had when the application started.
The device clock does not follow a change between daylight saving time and standard time, and the comparison
does not follow it either, so such a change of the server clock while the application runs does not affect
the correction. If the device clock or the server clock is set to a different time during a meet, the following
starts are measured against the old baseline. Restart the application in that case, since the baseline cannot tell
a changed clock from a delayed signal.

A start frame without a valid time of day starts the clock at its arrival.

## Simulator

The simulator replaces the timing device, so the overlays can be set up and shown without a device.
Its frames pass through a virtual serial port and take the same path as the data of a real device.
The simulator requires Linux or Windows.

With `TIMING_SIMULATE_EVENT` set to an event number of the loaded start list and no replayed recording,
it generates the heats of that event with random times in a loop. Every heat runs without disturbances
and in the regular order of load, start, splits, finish, and official frame.

With `TIMING_REPLAY_PATH` set to a [recording](#recording), such as `recordings/recording_2026-10-18.jsonl`,
it sends the recorded data with the pauses between the reads as they were recorded, including the pause of a restart.
Every pass, the first one included, begins after a pause of 10 seconds with the first data, so the time between
the start of the application and its first data is skipped. A clock that was set back between two lines
counts as no pause. The device clock jumps back with every pass and with every restart in the recording,
so the [start time correction](#start-time-correction) measures its baseline anew at these points.

With both settings, the simulator replays only the heats of that event from the recording. A heat spans the data
from the first byte of its first frame to the last byte of its last frame, including noise and the frames of other
events in between. The next load frame ends it, since the device then shows another heat, so a later frame
of the heat, such as a correction, is replayed with a short window of its own.
The pauses between the heats stay as they were recorded.

The replay uses the loaded meet data. If it differs from the meet files that were in effect when the replayed data
was recorded, a warning is logged and the overlays may show wrong or missing names. With a simulated event,
only what the overlays show of that event counts: the event with its distance, style, category, and phase,
its start entries, and their competitors. Upload the files of the recorded meet to replay it with its names.

## Recording

With `TIMING_RECORDING` on, which is the default, the application records the data of the timing device
in `recordings/` of the storage. Every day gets a file such as `recording_2026-10-18.jsonl`, named after the day
of the start of the application. A restart on the same day appends to the file of that day, a running application
keeps its file past midnight. A recording grows while it runs, so only the `local` storage driver records.
A replay is not recorded, the data of the simulator with random times is recorded and marked as such.

The recording never delays the overlays. If the disk cannot keep up, data is dropped, and the recording notes
how many bytes are missing at that point.

The config page lists the recordings. A recording can be uploaded, selected for the replay, downloaded as it is stored
or as a [table for analysis](#table-for-analysis), and deleted. An upload replaces a recording of the same name,
except the one being written. Neither the recording being written nor the replayed one can be deleted.

### Format

A recording is a [JSON Lines](https://jsonlines.org/) file. Every line is an object with a `type` and the time `at`
in ISO 8601 with the UTC offset of the server. There are four types of lines:

| Type      | Content                                                                                                                                                                                                            |
|-----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `session` | Start of the application, with the format version, the application version, the source, the port, and the baud rate.                                                                                               |
| `meet`    | The meet files in use from this point on, each with its name, its SHA-256 hash, and its content in the original encoding, base64 encoded. Written at the start and after every successful reload of the meet data. |
| `data`    | The bytes of one read from the device as hex bytes, at the time they arrived.                                                                                                                                      |
| `gap`     | The number of bytes dropped before this point.                                                                                                                                                                     |

The data lines hold the bytes as the device sent them, including noise and incomplete blocks.
If the application stopped while writing, the last line of a session can be incomplete.
It is skipped when the recording is read.

### Table for analysis

`swim-timer recording csv <recording> [file]` turns a recording into a CSV table. It decodes the data as the application
did while recording, with one row per session, meet line, frame, discarded block, and gap:

| Column                                             | Content                                                                                                              |
|----------------------------------------------------|----------------------------------------------------------------------------------------------------------------------|
| `Time`                                             | Time of the row. A frame is complete with the read that brought its last byte.                                       |
| `Session`                                          | Number of the session in the recording, counted from 1.                                                              |
| `Type`                                             | `session`, `meet`, `frame`, `discarded`, or `gap`.                                                                   |
| `Kind`                                             | Kind of a frame: `load`, `start`, `intermediate`, `finish`, `button`, `official`, or `unknown`.                      |
| `MsgType`, `MsgKind`, `TimeType`                   | The codes of the frame as the device sent them, see [Omega transmission protocol](./omega-transmission-protocol.md). |
| `Event`, `Heat`, `Lane`, `Lap`, `Rank`, `LapCount` | The values of the frame as the device counts them, heats from 1 and lane 10 as 0.                                    |
| `UsedLanes`                                        | The lanes in use, separated by spaces.                                                                               |
| `TimeMS`                                           | The time of the frame in milliseconds, for a start frame the device's time of day.                                   |
| `ClockOffsetMS`, `BaselineMS`, `DelayMS`           | The values of the [start time correction](#start-time-correction) of a start frame.                                  |
| `StartTime`                                        | The corrected start time of a start frame.                                                                           |
| `DroppedBytes`                                     | The number of bytes of a discarded block or a gap.                                                                   |
| `Hex`                                              | The bytes of a frame or a discarded block.                                                                           |
| `Details`                                          | The metadata of a session and the file names of a meet line.                                                         |

Every session starts with a new baseline of the start time correction, like a restart of the application.
Noise in front of a frame appears as a discarded block at the time the following header arrived.
Bytes still pending at the end of a session appear as a discarded block as well.
