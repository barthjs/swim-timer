import type { RecordResponse } from '@/api/types/record';
import type { RecordTableResponse } from '@/api/types/recordTable';
import { formatSwimtime } from '@/lib/live/time';
import { formatPlainDate } from '@/lib/ui/datetime';

/** The key of a column of a record, which the files of a record table name in their header. */
export type RecordColumn = Exclude<keyof RecordResponse, 'tableId'>;

/** The keys of the columns of a record in the order of a record table. */
export const RECORD_COLUMNS = [
  'course',
  'gender',
  'relayCount',
  'distance',
  'stroke',
  'ageFrom',
  'ageTo',
  'time',
  'approved',
  'name',
  'club',
  'place',
  'date',
  'birthYear',
] as const satisfies readonly RecordColumn[];

/** The distance of a record with the number of swimmers, which tells a relay from a leg. */
export type RecordDistance = Pick<RecordResponse, 'relayCount' | 'distance'>;

/** A record of the table of all records, with the table it belongs to. */
export type RecordRow = {
  // A key of the row that stays the same while the records do not change
  key: string;
  record: RecordResponse;
  // Undefined for a record whose table is not listed, which only happens while both lists are loading
  table: RecordTableResponse | undefined;
};

/** The filters of the table of all records. An empty list matches every value. */
export type RecordFilters = {
  query: string;
  courses: string[];
  genders: string[];
  strokes: string[];
  // The keys of recordDistanceKey, so a relay and an individual event over the length of its leg differ
  distances: string[];
};

/** Returns filters that match every record. */
export function emptyRecordFilters(): RecordFilters {
  return {
    query: '',
    courses: [],
    genders: [],
    strokes: [],
    distances: [],
  };
}

/** Formats a record time in hundredths of a second like the times of a heat. */
export function formatRecordTime(hundredths: number): string {
  return formatSwimtime(hundredths * 10);
}

/** Formats the day a record was set, or returns the value as it is if it is no YYYY-MM-DD date. */
export function formatRecordDate(date: string, locale: string): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return date;
  return formatPlainDate(date, locale);
}

/** Formats the distance of a record, with the number of legs of a relay. */
export function formatRecordDistance(record: RecordDistance): string {
  if (record.relayCount > 1) return `${record.relayCount}×${record.distance} m`;
  return `${record.distance} m`;
}

/** Returns the value the distance filter matches a record by. */
export function recordDistanceKey(record: RecordDistance): string {
  return `${record.relayCount}x${record.distance}`;
}

/** Joins the records with their tables, keyed by their table and their position in it. */
export function recordRows(records: RecordResponse[], tables: RecordTableResponse[]): RecordRow[] {
  const positions = new Map<string, number>();
  return records.map((record) => {
    const position = positions.get(record.tableId) ?? 0;
    positions.set(record.tableId, position + 1);
    return {
      key: `${record.tableId}-${position}`,
      record,
      table: tables.find((table) => table.id === record.tableId),
    };
  });
}

/** Reports whether the holder, the club, or the place of a record contains the search. */
export function matchesRecordSearch(record: RecordResponse, search: string): boolean {
  const term = search.trim().toLowerCase();
  if (!term) return true;
  return [record.name, record.club, record.place].join(' ').toLowerCase().includes(term);
}

/** Returns the rows that match the search and every set filter. */
export function filterRecords(rows: RecordRow[], filters: RecordFilters): RecordRow[] {
  return rows.filter(({ record }) => {
    if (filters.courses.length && !filters.courses.includes(record.course)) return false;
    if (filters.genders.length && !filters.genders.includes(record.gender)) return false;
    if (filters.strokes.length && !filters.strokes.includes(record.stroke)) return false;
    if (filters.distances.length && !filters.distances.includes(recordDistanceKey(record))) {
      return false;
    }
    return matchesRecordSearch(record, filters.query);
  });
}

/** Returns the distinct values of the rows for a filter, sorted. */
export function recordFacet<V extends string | number>(
  rows: RecordRow[],
  pick: (record: RecordResponse) => V,
): V[] {
  const values = [...new Set(rows.map(({ record }) => pick(record)))];
  return values.sort((a, b) =>
    typeof a === 'number' && typeof b === 'number' ? a - b : String(a).localeCompare(String(b)),
  );
}

/** Returns the distinct distances of the rows, the individual events before the relays, by length. */
export function recordDistances(rows: RecordRow[]): RecordDistance[] {
  const distances = new Map<string, RecordDistance>();
  for (const { record } of rows) {
    distances.set(recordDistanceKey(record), {
      relayCount: record.relayCount,
      distance: record.distance,
    });
  }
  return [...distances.values()].sort(
    (a, b) => a.relayCount - b.relayCount || a.distance - b.distance,
  );
}
