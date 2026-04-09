export enum WSMessageType {
  Init = 'init',
  Update = 'update',
}

export interface WSMessage {
  type: WSMessageType;
  data: RaceState;
}

export interface Lane {
  number: number;
  first_name: string;
  last_name: string;
  birthyear: string;
  club: string;
  lap: number;
  meter: number;
  time: number;
  rank: number;
  finished: boolean;
}

export enum RaceStatus {
  Load = 'load',
  Start = 'start',
  Official = 'official',
}

export interface RaceState {
  event: number;
  heat: number;
  max_heat: number;
  lap_count: number;
  name: string;
  status: RaceStatus;
  start_time?: string;
  lanes: Record<number, Lane>; // Sorted by lane number
}

export interface Config {
  config_file_path: string;
  log_level: string;
  pool_length: number;
  presets_dir: string;
  race_data_dir: string;
  serial_port: string;
  simulate_event: number;
  server: {
    host: string;
    port: number;
  };
  secret: string;
}

export interface Point {
  x: number;
  y: number;
}

export enum CornerKey {
  TopLeft = 'tl',
  TopRight = 'tr',
  BottomRight = 'br',
  BottomLeft = 'bl',
}

export interface LanePreset {
  [CornerKey.TopLeft]: Point;
  [CornerKey.TopRight]: Point;
  [CornerKey.BottomRight]: Point;
  [CornerKey.BottomLeft]: Point;
  locked?: boolean; // Locks the movement in the Editor
}

export enum TextAnchor {
  Left = 'left',
  Right = 'right',
}

export interface Preset {
  name: string;
  comment: string | null;
  virtualWidth: number;
  virtualHeight: number;
  reverseLanes: boolean; // true maps visual top lane to the highest race lane number
  textAnchor: TextAnchor; // layout direction based on camera position
  lanes: Record<number, LanePreset>;
}

export interface RaceDataFile {
  id: string;
  exists: boolean;
  isUploading: boolean;
}

export interface RaceDataUploadResult {
  filename: string;
  matched: boolean;
  error?: string;
}
