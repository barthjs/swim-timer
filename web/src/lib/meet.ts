import type {
  LiveHeat,
  MeetResponse,
  CompetitorResponse,
  EventResponse,
  StartEntryResponse,
} from '@/api/types';

/** A heat of the meet, identified by its event and its number counted from 1. */
export type MeetHeat = {
  event: EventResponse;
  heat: number;
  entries: MeetStartEntryRow[];
};

/** A start entry together with what it references, as listed in the tables. */
export type MeetStartEntryRow = {
  key: string;
  entry: StartEntryResponse;
  event: EventResponse;
  competitor: CompetitorResponse | null;
  // The position of the event in the program, see eventPositions.
  eventPosition: number;
  // Folded text of the competitor and the event, see foldText.
  searchText: string;
};

export type MeetClub = {
  name: string;
  competitorCount: number;
  startCount: number;
};

/** MeetIndex holds the meet with the lookups the overview needs, built once per loaded meet. */
export type MeetIndex = {
  events: EventResponse[];
  eventsByNumber: Map<number, EventResponse>;
  // The position of each event in the program, see eventPositions.
  positionsByEvent: Map<number, number>;
  competitorsById: Map<number, CompetitorResponse>;
  rows: MeetStartEntryRow[];
  rowsByEvent: Map<number, MeetStartEntryRow[]>;
  rowsByCompetitor: Map<number, MeetStartEntryRow[]>;
  // Every heat of the meet in schedule order: events in the order the meet lists them,
  // which is the order of the program, heats ascending.
  heats: MeetHeat[];
  clubs: MeetClub[];
};

/** A larger difference to the schedule means it belongs to another day and is not shown. */
const MAX_SCHEDULE_DELTA_MS = 12 * 60 * 60 * 1000;

export const EventState = {
  Done: 'done',
  Current: 'current',
  Upcoming: 'upcoming',
} as const;
export type EventState = (typeof EventState)[keyof typeof EventState];

export const EventKind = {
  All: 'all',
  Individual: 'individual',
  Relay: 'relay',
} as const;
export type EventKind = (typeof EventKind)[keyof typeof EventKind];

/** MeetFilters are the criteria of the overview. Empty lists do not restrict. */
export type MeetFilters = {
  query: string;
  days: string[];
  styles: string[];
  distances: string[];
  categories: string[];
  clubs: string[];
  kind: EventKind;
  upcomingOnly: boolean;
};

/** emptyFilters returns filters that match everything. */
export function emptyFilters(): MeetFilters {
  return {
    query: '',
    days: [],
    styles: [],
    distances: [],
    categories: [],
    clubs: [],
    kind: EventKind.All,
    upcomingOnly: false,
  };
}

/**
 * foldText prepares text for a search that ignores letter case and accents.
 */
export function foldText(text: string): string {
  return text
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .replace(/ß/g, 'ss')
    .toLowerCase();
}

/** eventText collects the searchable text of an event. */
function eventText(event: EventResponse): string {
  return [
    String(event.number),
    event.name,
    event.phaseName,
    event.distance?.name ?? '',
    event.style?.name ?? '',
    event.style?.abbreviation ?? '',
    event.category?.name ?? '',
  ].join(' ');
}

/** competitorText collects the searchable text of a competitor. */
function competitorText(competitor: CompetitorResponse | null): string {
  if (!competitor) return '';
  return [competitor.firstName, competitor.lastName, competitor.club, competitor.birthYear].join(
    ' ',
  );
}

/** pushTo appends a value to the list stored under key, creating the list if needed. */
function pushTo<K, V>(map: Map<K, V[]>, key: K, value: V): void {
  const list = map.get(key);
  if (list) {
    list.push(value);
    return;
  }
  map.set(key, [value]);
}

/**
 * eventPositions maps each event number to its position in the program, which is the order the meet lists the events.
 * Sorting by it follows the schedule, also for events without a scheduled time, such as finals.
 */
export function eventPositions(events: readonly EventResponse[]): Map<number, number> {
  return new Map(events.map((event, position) => [event.number, position]));
}

/**
 * buildMeetIndex links the start entries to their events and competitors. Entries of an event that is not part
 * of the meet are left out, since nothing could be shown about them. Entries of another phase than their event
 * are left out as well, as the live heat never lists them.
 */
export function buildMeetIndex(meet: MeetResponse): MeetIndex {
  const eventsByNumber = new Map(meet.events.map((event) => [event.number, event]));
  const positionsByEvent = eventPositions(meet.events);
  const competitorsById = new Map(
    meet.competitors.map((competitor) => [competitor.id, competitor]),
  );

  const rows: MeetStartEntryRow[] = [];
  const rowsByEvent = new Map<number, MeetStartEntryRow[]>();
  const rowsByCompetitor = new Map<number, MeetStartEntryRow[]>();

  for (const entry of meet.startEntries) {
    const event = eventsByNumber.get(entry.event);
    if (!event || entry.phase !== event.phase) continue;

    const competitor = competitorsById.get(entry.competitorId) ?? null;
    const row: MeetStartEntryRow = {
      key: `${entry.event}-${entry.heat}-${entry.lane}-${entry.relayPosition}-${entry.competitorId}`,
      entry,
      event,
      competitor,
      eventPosition: positionsByEvent.get(event.number) ?? 0,
      searchText: foldText(`${competitorText(competitor)} ${eventText(event)}`),
    };

    rows.push(row);
    pushTo(rowsByEvent, entry.event, row);
    pushTo(rowsByCompetitor, entry.competitorId, row);
  }

  const heats: MeetHeat[] = [];
  for (const event of meet.events) {
    const eventRows = rowsByEvent.get(event.number) ?? [];
    for (let heat = 1; heat <= event.heatCount; heat++) {
      heats.push({
        event,
        heat,
        entries: eventRows.filter((row) => row.entry.heat === heat),
      });
    }
  }

  return {
    events: meet.events,
    eventsByNumber,
    positionsByEvent,
    competitorsById,
    rows,
    rowsByEvent,
    rowsByCompetitor,
    heats,
    clubs: buildClubs(meet.competitors, rowsByCompetitor),
  };
}

/** buildClubs groups the competitors by club with their start entries. */
function buildClubs(
  competitors: CompetitorResponse[],
  rowsByCompetitor: Map<number, MeetStartEntryRow[]>,
): MeetClub[] {
  const clubs = new Map<string, MeetClub>();

  for (const competitor of competitors) {
    const name = competitor.club.trim();
    if (!name) continue;

    const club = clubs.get(name) ?? {
      name,
      competitorCount: 0,
      startCount: 0,
    };
    club.competitorCount++;
    club.startCount += rowsByCompetitor.get(competitor.id)?.length ?? 0;
    clubs.set(name, club);
  }

  return [...clubs.values()].sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * currentHeatIndex returns the position of the live heat in the schedule, or -1 without one.
 * A heat beyond the heat count of its event, which happens when more heats are run than planned,
 * counts as the last heat of the event.
 */
export function currentHeatIndex(index: MeetIndex, live: LiveHeat | null): number {
  if (!live || !live.status || live.heat < 1) return -1;

  let last = -1;
  for (const [i, heat] of index.heats.entries()) {
    if (heat.event.number !== live.event) continue;
    if (heat.heat === live.heat) return i;
    last = i;
  }

  const event = index.eventsByNumber.get(live.event);
  if (event && live.heat > event.heatCount) return last;
  return -1;
}

/**
 * eventState tells whether an event lies before, at, or after the live heat in the schedule.
 * Without a live heat in the schedule, the state is unknown. An event without heats has not been swum,
 * so it counts as upcoming wherever it stands in the program.
 */
export function eventState(
  index: MeetIndex,
  event: EventResponse,
  currentIndex: number,
): EventState | null {
  if (currentIndex < 0) return null;

  const current = index.heats[currentIndex];
  if (!current) return null;
  if (current.event.number === event.number) return EventState.Current;

  const first = index.heats.findIndex((heat) => heat.event.number === event.number);
  if (first < 0) return EventState.Upcoming;
  return first < currentIndex ? EventState.Done : EventState.Upcoming;
}

/** nextHeats returns the heats that follow the live heat in the schedule. */
export function nextHeats(index: MeetIndex, currentIndex: number, count: number): MeetHeat[] {
  if (currentIndex < 0) return [];
  return index.heats.slice(currentIndex + 1, currentIndex + 1 + count);
}

/**
 * scheduledTime returns the scheduled start of an event as a local date,
 * or null if date or time are missing or cannot be read.
 */
export function scheduledTime(event: EventResponse): Date | null {
  const date = /^(\d{4})-(\d{2})-(\d{2})$/.exec(event.date);
  const time = /^(\d{1,2}):(\d{2})$/.exec(event.time.trim());
  if (!date || !time) return null;

  return new Date(
    Number(date[1]),
    Number(date[2]) - 1,
    Number(date[3]),
    Number(time[1]),
    Number(time[2]),
  );
}

/**
 * scheduleDelta estimates how far the meet runs behind its schedule, in milliseconds, negative if ahead.
 * Only events have a scheduled time, so the expected start of the live heat is interpolated
 * between the scheduled start of its event and of the next event, assuming the heats of an event take equally long.
 * It returns null if the schedule does not cover the live heat,
 * or if the difference is so large that the schedule belongs to another day.
 */
export function scheduleDelta(index: MeetIndex, currentIndex: number, now: Date): number | null {
  const current = index.heats[currentIndex];
  if (!current) return null;

  const start = scheduledTime(current.event);
  if (!start) return null;

  let expected = start.getTime();
  const following = index.heats
    .slice(currentIndex + 1)
    .find((heat) => heat.event.number !== current.event.number);
  const next = following ? scheduledTime(following.event) : null;

  if (next && next.getTime() > start.getTime() && current.event.heatCount > 0) {
    const share = (current.heat - 1) / current.event.heatCount;
    expected += share * (next.getTime() - start.getTime());
  }

  const delta = now.getTime() - expected;
  return Math.abs(delta) > MAX_SCHEDULE_DELTA_MS ? null : delta;
}

/**
 * matchesAny reports whether value is one of the selected values, or whether nothing is selected.
 */
function matchesAny(selected: string[], value: string | undefined): boolean {
  return selected.length === 0 || (value !== undefined && selected.includes(value));
}

/**
 * matchesEventFilters checks the criteria that belong to an event, everything except the text query and clubs.
 */
export function matchesEventFilters(
  event: EventResponse,
  filters: MeetFilters,
  state: EventState | null,
): boolean {
  if (!matchesAny(filters.days, event.date)) return false;
  if (!matchesAny(filters.styles, event.style?.name)) return false;
  if (!matchesAny(filters.distances, event.distance?.name)) return false;
  if (!matchesAny(filters.categories, event.category?.name)) return false;

  const relay = (event.distance?.relayCount ?? 1) > 1;
  if (filters.kind === EventKind.Individual && relay) return false;
  if (filters.kind === EventKind.Relay && !relay) return false;

  if (filters.upcomingOnly && state === EventState.Done) return false;
  return true;
}

/** queryTerms splits a search query into folded words. */
function queryTerms(query: string): string[] {
  return foldText(query).split(/\s+/).filter(Boolean);
}

/** matchesTerms reports whether text contains every term. */
function matchesTerms(text: string, terms: string[]): boolean {
  return terms.every((term) => text.includes(term));
}

/** matchesRow reports whether a start entry matches the club filter and the query. */
function matchesRow(row: MeetStartEntryRow, filters: MeetFilters, terms: string[]): boolean {
  if (!matchesAny(filters.clubs, row.competitor?.club)) return false;
  return matchesTerms(row.searchText, terms);
}

/**
 * filterRows returns the start entries that match every criterion.
 * Every word of the query has to appear in the competitor or the event.
 */
export function filterRows(
  index: MeetIndex,
  filters: MeetFilters,
  stateOf: (event: EventResponse) => EventState | null,
): MeetStartEntryRow[] {
  const terms = queryTerms(filters.query);
  return index.rows.filter(
    (row) =>
      matchesEventFilters(row.event, filters, stateOf(row.event)) &&
      matchesRow(row, filters, terms),
  );
}

/**
 * filterEvents returns the events that match the event criteria and, with a query or a club filter,
 * either match the query themselves or have a matching start entry.
 */
export function filterEvents(
  index: MeetIndex,
  filters: MeetFilters,
  stateOf: (event: EventResponse) => EventState | null,
): EventResponse[] {
  const terms = queryTerms(filters.query);
  const restrictsEntries = terms.length > 0 || filters.clubs.length > 0;

  return index.events.filter((event) => {
    if (!matchesEventFilters(event, filters, stateOf(event))) return false;
    if (!restrictsEntries) return true;

    if (filters.clubs.length === 0 && matchesTerms(foldText(eventText(event)), terms)) return true;
    const rows = index.rowsByEvent.get(event.number) ?? [];
    return rows.some((row) => matchesRow(row, filters, terms));
  });
}

/** filterClubs returns the clubs whose name matches the query. */
export function filterClubs(index: MeetIndex, filters: MeetFilters): MeetClub[] {
  const terms = queryTerms(filters.query);
  return index.clubs.filter(
    (club) => matchesAny(filters.clubs, club.name) && matchesTerms(foldText(club.name), terms),
  );
}

/**
 * facetValues collects the distinct, sorted values of one property of the events, for the filter menus.
 */
export function facetValues(
  events: EventResponse[],
  value: (event: EventResponse) => string | undefined,
): string[] {
  const values = new Set<string>();
  for (const event of events) {
    const item = value(event);
    if (item) values.add(item);
  }
  return [...values].sort((a, b) =>
    a.localeCompare(b, undefined, {
      numeric: true,
    }),
  );
}

/** eventName names an event together with its phase, if the phase has a name. */
export function eventName(event: Pick<EventResponse, 'name' | 'phaseName'>): string {
  return [event.name.trim(), event.phaseName.trim()].filter(Boolean).join(' · ');
}

/** competitorName writes a competitor as "Lastname, Firstname". */
export function competitorName(competitor: CompetitorResponse | null): string {
  if (!competitor) return '';
  return [competitor.lastName, competitor.firstName].filter(Boolean).join(', ');
}

/**
 * formatDay writes an ISO date as a short weekday and date in the given locale,
 * or returns the value unchanged if it is not an ISO date.
 */
export function formatDay(date: string, locale: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!match) return date;

  const day = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
  return new Intl.DateTimeFormat(locale, {
    weekday: 'short',
    day: '2-digit',
    month: '2-digit',
  }).format(day);
}
