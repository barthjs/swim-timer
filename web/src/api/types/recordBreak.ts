import type { RecordResponse } from '@/api/types/record';

/** A record set or equalled at the meet, with the record it broke. */
export type RecordBreakResponse = {
  id: string;
  tableId: string;
  tableName: string;
  abbreviation: string;
  event: number;
  heat: number;
  lane: number;
  equalled: boolean;
  record: RecordResponse;
  previous: RecordResponse;
  brokenAt: string;
};
