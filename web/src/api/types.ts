export type AuthLoginRequest = {
  password: string;
};

export type AuthStatusResponse = {
  authenticated: boolean;
};

export type AppSettings = {
  name: string;
  env: string;
  language: string;
  secret: string;
};

export type LogSettings = {
  level: string;
  channel: string;
  path: string;
  format: string;
  maxSizeMb: number;
  maxAgeDays: number;
  compress: boolean;
};

export type ServerSettings = {
  host: string;
  port: number;
  trustedProxies: string;
  corsAllowedOrigins: string[];
  corsAllowCredentials: boolean;
  sslEnabled: boolean;
  sslCertFile: string;
  sslKeyFile: string;
};

export type S3Settings = {
  endpoint: string;
  region: string;
  bucket: string;
  accessKey: string;
  secretKey: string;
  usePathStyle: boolean;
};

export type StorageSettings = {
  driver: 'local' | 's3';
  path: string;
  s3: S3Settings;
};

export type ThemeSettings = {
  primary: string;
  secondary: string;
  success: string;
  info: string;
  warning: string;
  error: string;
  neutral: string;
};

export type TimingSettings = {
  serialPort: string;
  baudRate: number;
  poolLength: number;
  simulateEvent: number;
  replayPath: string;
  recording: boolean;
};

/**
 * ConfigFile is the config file a save writes to. path is empty if none was found at startup,
 * a save then creates one. exists is false until the file at path is created.
 */
export type ConfigFile = {
  path: string;
  exists: boolean;
};

/**
 * ConfigSource is where the value of a setting comes from. A flag or an environment variable
 * overrides the config file. name is the flag, the variable, or the file path, empty for the default.
 */
export type ConfigSource = {
  kind: 'default' | 'file' | 'env' | 'flag';
  name: string;
};

/** ConfigSettings holds the sections that ConfigResponse and ConfigSaveRequest share. */
export type ConfigSettings = {
  app: AppSettings;
  log: LogSettings;
  server: ServerSettings;
  storage: StorageSettings;
  theme: ThemeSettings;
  timing: TimingSettings;
};

/**
 * ConfigSaveRequest is the configuration to save. A secret sent empty keeps its saved value,
 * unless its path is listed in clearSecrets.
 */
export type ConfigSaveRequest = ConfigSettings & {
  clearSecrets: string[];
};

/**
 * ConfigResponse is the configuration without its secrets. secrets reports by path whether each secret is set,
 * since the server never sends their values. defaults holds the default of every setting,
 * sources reports by path where each setting comes from.
 */
export type ConfigResponse = ConfigSettings & {
  secrets: Record<string, boolean>;
  defaults: ConfigSettings;
  sources: Record<string, ConfigSource>;
  file: ConfigFile;
};

export type ConfigTestResponse = {
  ok: boolean;
  message?: string;
};

/** HeatStatus is what the timing device last announced for the live heat. */
export const HeatStatus = {
  Load: 'load',
  Start: 'start',
  Official: 'official',
} as const;
export type HeatStatus = (typeof HeatStatus)[keyof typeof HeatStatus];

/** Phase is the phase of an event as the timing software numbers it, see LiveHeat.phase. */
export const Phase = {
  TimedFinal: 0,
  Heats: 1,
  Semifinal: 2,
  Final: 3,
  SwimOff: 4,
  Reswim: 5,
} as const;
export type Phase = (typeof Phase)[keyof typeof Phase];

/**
 * Lane is one lane of the live heat. A lane without a start list entry,
 * which only has a time, has empty names, birth year, and club.
 */
export type Lane = {
  number: number;
  firstName: string;
  lastName: string;
  birthYear: string;
  club: string;
  // lap, meter, time, and rank belong to the highest lap the lane has reached.
  lap: number;
  meter: number;
  // Heat time in milliseconds, 0 without a time.
  time: number;
  // Rank within the lap, equal times share a rank. 0 while the lane has no rank yet.
  rank: number;
  // Set once the device sent a finish time for the lane.
  finished: boolean;
};

/** LiveHeat is the heat the timing device currently reports, with one entry per lane keyed by lane number. */
export type LiveHeat = {
  event: number;
  // A phase number of the timing software, see Phase. Other values may occur.
  phase: number;
  // The name of the phase from the meet data, empty if it is unknown.
  phaseName: string;
  heat: number;
  heatCount: number;
  lapCount: number;
  name: string;
  // Empty until the first frame of the timing device arrives.
  status: HeatStatus | '';
  startTime?: string;
  // Heat time in milliseconds at which the timer stops, the highest finish time once every lane the device
  // reports in use has finished. Absent while one of them is still swimming.
  stopTime?: number;
  // The lowest lane number of the pool, 0 or 1, derived from the start list of the meet.
  firstLane: number;
  lanes: Record<string, Lane>;
};

/** LiveMessageType names what a websocket message announces. */
export const LiveMessageType = {
  Init: 'init',
  Update: 'update',
  // The meet data was reloaded. It carries no data, pages showing the meet fetch it again.
  Meet: 'meet',
} as const;
export type LiveMessageType = (typeof LiveMessageType)[keyof typeof LiveMessageType];

/** LiveMessage is a message of the live websocket. */
export type LiveMessage =
  | {
      type: typeof LiveMessageType.Init | typeof LiveMessageType.Update;
      // The server clock when the message was sent, as an RFC 3339 timestamp.
      serverTime: string;
      // Always the full live heat, never null. Before the first frame its status is empty.
      data: LiveHeat;
    }
  | {
      type: typeof LiveMessageType.Meet;
      serverTime: string;
    };

export type DistanceResponse = {
  name: string;
  // Total length in meters, all relay legs together.
  length: number;
  // Swimmers per team, 1 for an individual event.
  relayCount: number;
};

export type StyleResponse = {
  name: string;
  abbreviation: string;
};

export type CategoryResponse = {
  name: string;
  abbreviation: string;
};

/** EventResponse is one event of the meet. The events of a meet are listed in the order of the program. */
export type EventResponse = {
  number: number;
  // A phase number of the timing software, see Phase.
  phase: number;
  // The name of the phase from the meet data, empty if it is unknown.
  phaseName: string;
  heatCount: number;
  name: string;
  // ISO date (YYYY-MM-DD), or the exported value if it could not be read, empty if unknown.
  date: string;
  // Scheduled start time as exported, such as "16:00", empty if the event has no scheduled time.
  time: string;
  distance: DistanceResponse | null;
  style: StyleResponse | null;
  category: CategoryResponse | null;
};

/** StartEntryResponse is one start of a competitor in a heat. */
export type StartEntryResponse = {
  event: number;
  phase: number;
  // Counted from 1, like the timing device.
  heat: number;
  lane: number;
  relayPosition: number;
  competitorId: number;
};

export type CompetitorResponse = {
  id: number;
  firstName: string;
  lastName: string;
  birthYear: string;
  club: string;
  category: string;
};

/** MeetResponse is the loaded meet data. loadedAt is missing while no meet data is loaded. */
export type MeetResponse = {
  loadedAt?: string;
  events: EventResponse[];
  startEntries: StartEntryResponse[];
  competitors: CompetitorResponse[];
};

/** MeetFile names the export files of the timing software the meet data is read from. */
export const MeetFile = {
  Categories: 'LSTCAT',
  Competitors: 'LSTCONC',
  Distances: 'LSTLONG',
  Events: 'LSTRACE',
  Phases: 'LSTROUND',
  StartEntries: 'LSTSTART',
  Styles: 'LSTSTYLE',
} as const;
export type MeetFile = (typeof MeetFile)[keyof typeof MeetFile];

/** MeetFileResponse reports whether a meet file is stored. */
export type MeetFileResponse = {
  file: MeetFile;
  // The name the file is stored under, including the extension it is read with. Missing while unavailable.
  filename?: string;
  available: boolean;
};

/** MeetFileUploadResponse is the result for one uploaded file. error names why it was not stored. */
export type MeetFileUploadResponse = {
  filename: string;
  file?: MeetFile;
  error?: string;
};

export type MetaResponse = {
  name: string;
  version: string;
  language: string;
  theme: ThemeSettings;
};

export const TextAnchor = {
  Left: 'left',
  Right: 'right',
} as const;
export type TextAnchor = (typeof TextAnchor)[keyof typeof TextAnchor];

export const CornerKey = {
  TopLeft: 'tl',
  TopRight: 'tr',
  BottomRight: 'br',
  BottomLeft: 'bl',
} as const;
export type CornerKey = (typeof CornerKey)[keyof typeof CornerKey];

declare const fractionBrand: unique symbol;

/**
 * Fraction is a share of the width or height of the output image, from 0 to 1.
 * The brand keeps the stored fractions apart from pixel values, which are plain numbers.
 */
export type Fraction = number & { readonly [fractionBrand]: true };

/**
 * PresetPointRequest is a position in the output image as fractions of its width and height,
 * from 0, 0 at the top left to 1, 1 at the bottom right.
 */
export type PresetPointRequest = {
  x: Fraction;
  y: Fraction;
};

export type PresetLaneRequest = Record<CornerKey, PresetPointRequest> & {
  locked: boolean;
};

/**
 * PresetPointResponse is a position in the output image as fractions of its width and height,
 * from 0, 0 at the top left to 1, 1 at the bottom right.
 */
export type PresetPointResponse = {
  x: Fraction;
  y: Fraction;
};

export type PresetLaneResponse = Record<CornerKey, PresetPointResponse> & {
  locked: boolean;
};

/** PresetSaveRequest is a preset to create or replace. The name has to match the name in the path. */
export type PresetSaveRequest = {
  name: string;
  comment: string | null;
  frameWidth: number;
  frameHeight: number;
  textSize: number;
  laneGap: number;
  reverseLanes: boolean;
  textAnchor: TextAnchor;
  lanes: Record<string, PresetLaneRequest>;
};

export type PresetResponse = {
  name: string;
  comment: string | null;
  // The reference resolution of the output image. It sets the aspect ratio the editor draws the preset in
  // and the pixels it shows, while the lanes are stored as fractions.
  frameWidth: number;
  frameHeight: number;
  // The height of the text in percent of the height of a lane.
  textSize: number;
  // The gap between two neighboring lanes in percent of the height of a lane, which aligning keeps.
  laneGap: number;
  reverseLanes: boolean;
  textAnchor: TextAnchor;
  lanes: Record<string, PresetLaneResponse>;
};

export type RecordingResponse = {
  name: string;
  // The storage path the replay is configured with.
  path: string;
  // Whether the recording is being written right now.
  current: boolean;
};

export type TimingStatusResponse = {
  connected: boolean;
  // The open serial port, empty while disconnected.
  port: string;
  // The configured port, empty if every port found is tried.
  configuredPort: string;
  // The event the simulator plays, 0 when a real device is read or the whole recording is replayed.
  simulateEvent: number;
  // The name of the recording the simulator replays, empty when no recording is replayed.
  replayFile: string;
};

/**
 * UpdateCheckResponse compares the running version with the latest release.
 * unsupported names why this installation cannot install it itself, missing if it can.
 */
export type UpdateCheckResponse = {
  current: string;
  latest: string;
  url: string;
  newer: boolean;
  unsupported?: 'development_build' | 'container' | 'package_managed' | 'not_writable';
};

export type UpdateStatusResponse = {
  state: 'idle' | 'running' | 'failed' | 'applied';
  version?: string;
  error?: string;
};
