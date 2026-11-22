export type TimingStatusResponse = {
  connected: boolean;
  // The open serial port, empty while disconnected
  port: string;
  // The configured port, empty if every port found is tried
  configuredPort: string;
  // The event the simulator plays, 0 when a real device is read or the whole recording is replayed
  simulateEvent: number;
  // The name of the recording the simulator replays, empty when no recording is replayed
  replayFile: string;
};
