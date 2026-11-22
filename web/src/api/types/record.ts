/** A record of a record table. Times are in hundredths of a second. */
export type RecordResponse = {
  tableId: string;
  course: 'SCM' | 'LCM';
  gender: string;
  // The number of swimmers per team, 1 for an individual event
  relayCount: number;
  // The distance each swimmer swims in meters, a relay leg for a relay
  distance: number;
  stroke: string;
  // The ages the record applies to, 0 for an open bound
  ageFrom: number;
  ageTo: number;
  time: number;
  approved: boolean;
  name: string;
  club: string;
  place: string;
  // The day the record was set as YYYY-MM-DD, empty if unknown
  date: string;
  // 0 if unknown
  birthYear: number;
};

/** Every record of every record table. loadedAt is missing before the tables are loaded. */
export type RecordListResponse = {
  loadedAt?: string;
  records: RecordResponse[];
};
