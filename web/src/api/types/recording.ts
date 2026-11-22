export type RecordingResponse = {
  name: string;
  // The storage path the replay is configured with
  path: string;
  // Whether the recording is being written right now
  current: boolean;
};
