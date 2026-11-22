export type DistanceResponse = {
  name: string;
  // Total length in meters, all relay legs together
  length: number;
  // Swimmers per team, 1 for an individual event
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

/** One event of the meet. The events of a meet are listed in the order of the program. */
export type EventResponse = {
  number: number;
  // A phase number of the timing software, see Phase
  phase: number;
  // The name of the phase from the meet data, empty if it is unknown
  phaseName: string;
  heatCount: number;
  name: string;
  // ISO date (YYYY-MM-DD), or the exported value if it could not be read, empty if unknown
  date: string;
  // Scheduled start time as exported, such as "16:00", empty if the event has no scheduled time
  time: string;
  distance: DistanceResponse | null;
  style: StyleResponse | null;
  category: CategoryResponse | null;
};

/** One start of a competitor in a heat. */
export type StartEntryResponse = {
  event: number;
  phase: number;
  // Counted from 1, like the timing device
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

/** The loaded meet data. loadedAt is missing while no meet data is loaded. */
export type MeetResponse = {
  loadedAt?: string;
  events: EventResponse[];
  startEntries: StartEntryResponse[];
  competitors: CompetitorResponse[];
};
