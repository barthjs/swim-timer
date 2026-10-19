---
title: Connecting the timing system
description: Connect the Omega timing system over a serial port or a serial-over-IP bridge.
sidebar:
  order: 1
---

Swim-Timer reads the data of the Omega timing system from Swiss Timing over its serial output, as described in the
[Omega transmission protocol](../reference/omega-transmission-protocol.md). The output is usually connected
to the computer with a USB serial adapter.

## Serial port

The Timing tab of the configuration page sets the connection:

| Setting     | Value                                                               |
| ----------- | ------------------------------------------------------------------- |
| Serial port | The port of the timing system. Empty tries every serial port found. |
| Baud rate   | The baud rate set on the timing system, 9600 by default.            |

Without a configured port, Swim-Timer tries every serial port it finds and uses the first one that opens.
On a computer with several serial ports, set the port of the timing system explicitly.

On Linux, only members of the `dialout` group may open serial ports. The services of the Debian package
and of the system-wide installation get that group themselves. A user who runs Swim-Timer as a desktop application
or as a user service needs to be added to the group and log in again:

```sh
sudo usermod -aG dialout "$USER"
```

## Container

In a container, the serial port has to be passed in with `devices` in the compose file:

```yaml
services:
  swim-timer:
    devices:
      - /dev/ttyUSB0:/dev/ttyUSB0
```

The container makes the passed ports readable for Swim-Timer when it starts.

## Serial-over-IP bridge

A device server or a computer running ser2net makes the serial output reachable over the network.
With the serial port set to its `host:port`, Swim-Timer connects to the bridge instead of opening a local port.

## Connection status

The dashboard shows whether the timing system is connected and through which port. A dropped connection is opened
again every few seconds. The overlays keep showing the last heat while the timing system is disconnected,
and its data in that time is lost.

## Pool length

The pool length gives the distance a swimmer has covered at each lap. It is 50 m by default. With 0, Swim-Timer
derives it from the distance of the event and the number of laps, see
[Distance and pool length](../concepts/live-heat.md#distance-and-pool-length).

## Time zone

The computer running Swim-Timer has to use the time zone of the timing system, since Swim-Timer compares
both clocks to [correct the start time](../concepts/start-time-correction.md) of heats that are loaded late.
In a container, set `TZ` in the `.env` file.
