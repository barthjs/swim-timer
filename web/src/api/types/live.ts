/** What the timing device last announced for the live heat. */
export const HeatStatus = {
  Load: 'load',
  Start: 'start',
  Official: 'official',
} as const;
export type HeatStatus = (typeof HeatStatus)[keyof typeof HeatStatus];

/** The phase of an event as the timing software numbers it, see LiveHeat.phase. */
export const Phase = {
  TimedFinal: 0,
  Heats: 1,
  Semifinal: 2,
  Final: 3,
  SwimOff: 4,
  Reswim: 5,
} as const;
export type Phase = (typeof Phase)[keyof typeof Phase];

/** A record a lane of the live heat broke or equalled. */
export type LaneRecord = {
  abbreviation: string;
  tableName: string;
  // The ages the record applies to, 0 for an open bound
  ageFrom: number;
  ageTo: number;
  equalled: boolean;
  // The time of the record before in milliseconds, like the times of the lanes, and its holder
  previousTime: number;
  previousName: string;
};

/**
 * One lane of the live heat. A lane without a start list entry,
 * which only has a time, has empty names, birth year, and club.
 */
export type Lane = {
  number: number;
  firstName: string;
  lastName: string;
  birthYear: string;
  club: string;
  // lap, meter, time, and rank belong to the highest lap the lane has reached
  lap: number;
  meter: number;
  // Heat time in milliseconds, 0 without a time
  time: number;
  // Rank within the lap, equal times share a rank. 0 while the lane has no rank yet.
  rank: number;
  // Set once the device sent a finish time for the lane
  finished: boolean;
  // The records the lane broke or equalled, set once the lane finished
  records?: LaneRecord[];
};

/** The heat the timing device currently reports, with one entry per lane keyed by lane number. */
export type LiveHeat = {
  event: number;
  // A phase number of the timing software, see Phase. Other values may occur.
  phase: number;
  // The name of the phase from the meet data, empty if it is unknown
  phaseName: string;
  heat: number;
  heatCount: number;
  lapCount: number;
  name: string;
  // Empty until the first frame of the timing device arrives
  status: HeatStatus | '';
  startTime?: string;
  // Heat time in milliseconds at which the timer stops,
  // the highest finish time once every lane the device reports in use has finished.
  // Absent while one of them is still swimming.
  stopTime?: number;
  // The lowest lane number of the pool, 0 or 1, derived from the start list of the meet
  firstLane: number;
  lanes: Record<string, Lane>;
};

/** Names what a websocket message announces. */
export const LiveMessageType = {
  Init: 'init',
  Update: 'update',
  // The meet data was reloaded. It carries no data, pages showing the meet fetch it again.
  Meet: 'meet',
} as const;
export type LiveMessageType = (typeof LiveMessageType)[keyof typeof LiveMessageType];

export type LiveMessage = {
  type: LiveMessageType;
  // The server clock when the message was sent, as an RFC 3339 timestamp
  serverTime: string;
  // Always the full live heat, never null. Before the first frame its status is empty.
  data: LiveHeat;
};
