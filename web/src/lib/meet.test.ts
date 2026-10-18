import { describe, expect, it } from 'vitest';

import type { LiveHeat, MeetResponse, EventResponse } from '@/api/types';
import { sortRows } from '@/lib/table/view';
import {
  EventKind,
  EventState,
  buildMeetIndex,
  currentHeatIndex,
  emptyFilters,
  eventName,
  eventPositions,
  eventState,
  filterClubs,
  filterEvents,
  filterRows,
  foldText,
  nextHeats,
  scheduleDelta,
  scheduledTime,
} from '@/lib/meet';

/** event builds an individual event with the given overrides. */
function event(number: number, overrides: Partial<EventResponse> = {}): EventResponse {
  return {
    number,
    phase: 0,
    phaseName: '',
    heatCount: 2,
    name: `${number}`,
    date: '2026-05-01',
    time: '10:00',
    distance: {
      name: '100 m',
      length: 100,
      relayCount: 1,
    },
    style: {
      name: 'Breaststroke',
      abbreviation: 'BR',
    },
    category: {
      name: 'Open',
      abbreviation: 'O',
    },
    ...overrides,
  };
}

const meet: MeetResponse = {
  events: [
    event(1, {
      name: '100 m Breaststroke Open',
      time: '10:00',
    }),
    event(2, {
      name: '4x50 m Freestyle Open',
      time: '11:00',
      heatCount: 1,
      distance: {
        name: '4x50 m',
        length: 200,
        relayCount: 4,
      },
      style: {
        name: 'Freestyle',
        abbreviation: 'FR',
      },
    }),
  ],
  startEntries: [
    {
      event: 1,
      phase: 0,
      heat: 1,
      lane: 3,
      relayPosition: 0,
      competitorId: 10,
    },
    {
      event: 1,
      phase: 0,
      heat: 2,
      lane: 4,
      relayPosition: 0,
      competitorId: 11,
    },
    {
      event: 2,
      phase: 0,
      heat: 1,
      lane: 5,
      relayPosition: 0,
      competitorId: 10,
    },
    {
      event: 99,
      phase: 0,
      heat: 1,
      lane: 1,
      relayPosition: 0,
      competitorId: 11,
    },
  ],
  competitors: [
    {
      id: 10,
      firstName: 'First',
      lastName: 'Accénted',
      birthYear: '2010',
      club: 'Club Two',
      category: 'O',
    },
    {
      id: 11,
      firstName: 'First',
      lastName: 'Plain',
      birthYear: '2011',
      club: 'Club One',
      category: 'O',
    },
  ],
};

/** live builds a started live heat of an event. */
function live(eventNumber: number, heat: number): LiveHeat {
  return {
    event: eventNumber,
    phase: 0,
    phaseName: '',
    heat,
    heatCount: 2,
    lapCount: 2,
    name: '',
    status: 'start',
    firstLane: 1,
    lanes: {},
  };
}

const index = buildMeetIndex(meet);
const noState = () => null;

describe('foldText', () => {
  it('ignores letter case and accents', () => {
    const folded = foldText('ÄÖÜ ß É');

    expect(folded).toBe('aou ss e');
  });
});

describe('eventPositions', () => {
  it('returns the program position of every event, also without a scheduled time', () => {
    const events = [
      event(1, {
        time: '09:00',
      }),
      event(2, {
        time: '',
      }),
      event(3, {
        time: '10:00',
      }),
    ];

    const positions = eventPositions(events);

    const rows = [...events].reverse().map((item) => ({
      number: item.number,
      schedule: positions.get(item.number) ?? 0,
    }));
    const sorted = sortRows(
      rows,
      {
        id: 'schedule',
        desc: false,
      },
      (row) => row.schedule,
    );
    expect(sorted.map((row) => row.number)).toEqual([1, 2, 3]);
  });
});

describe('buildMeetIndex', () => {
  it('returns the start entries of every competitor', () => {
    const built = buildMeetIndex(meet);

    expect(built.rowsByCompetitor.get(10)).toHaveLength(2);
  });

  it('lists every heat in schedule order', () => {
    const built = buildMeetIndex(meet);

    expect(built.heats.map((heat) => `${heat.event.number}/${heat.heat}`)).toEqual([
      '1/1',
      '1/2',
      '2/1',
    ]);
    expect(built.heats[1]?.entries.map((row) => row.entry.lane)).toEqual([4]);
  });

  it('keeps the order of the events as the meet lists them', () => {
    const source = {
      ...meet,
      events: [meet.events[1]!, meet.events[0]!],
    };

    const reordered = buildMeetIndex(source);

    expect(reordered.heats.map((heat) => `${heat.event.number}/${heat.heat}`)).toEqual([
      '2/1',
      '1/1',
      '1/2',
    ]);
  });

  it('gives every start entry the program position of its event', () => {
    const source = {
      ...meet,
      events: [meet.events[1]!, meet.events[0]!],
    };

    const reordered = buildMeetIndex(source);

    expect(reordered.rows.map((row) => `${row.event.number}/${row.eventPosition}`)).toEqual([
      '1/1',
      '1/1',
      '2/0',
    ]);
  });

  it('counts competitors and starts per club', () => {
    const built = buildMeetIndex(meet);

    expect(built.clubs).toEqual([
      {
        name: 'Club One',
        competitorCount: 1,
        startCount: 1,
      },
      {
        name: 'Club Two',
        competitorCount: 1,
        startCount: 2,
      },
    ]);
  });

  it('leaves out entries of unknown events', () => {
    const built = buildMeetIndex(meet);

    expect(built.rows).toHaveLength(3);
  });

  it('leaves out entries of another phase than their event', () => {
    const source = {
      ...meet,
      startEntries: [
        ...meet.startEntries,
        {
          event: 1,
          phase: 3,
          heat: 1,
          lane: 6,
          relayPosition: 0,
          competitorId: 11,
        },
      ],
    };

    const otherPhase = buildMeetIndex(source);

    expect(otherPhase.rows).toEqual(index.rows);
    expect(otherPhase.rowsByEvent.get(1)).toHaveLength(2);
    expect(otherPhase.rowsByCompetitor.get(11)).toHaveLength(1);
    expect(otherPhase.heats[0]?.entries.map((row) => row.entry.lane)).toEqual([3]);
    expect(otherPhase.clubs.find((club) => club.name === 'Club One')?.startCount).toBe(1);
  });
});

describe('currentHeatIndex', () => {
  it('returns the position of the live heat', () => {
    const heat = live(1, 2);

    const current = currentHeatIndex(index, heat);

    expect(current).toBe(1);
  });

  it('counts a heat beyond the heat count as the last heat of its event', () => {
    const heat = live(1, 3);

    const current = currentHeatIndex(index, heat);

    expect(current).toBe(1);
  });

  it.each<{ name: string; heat: LiveHeat | null }>([
    {
      name: 'without a live heat',
      heat: null,
    },
    {
      name: 'for a heat that has not started',
      heat: {
        ...live(1, 1),
        status: '',
      },
    },
    {
      name: 'for heat zero',
      heat: live(1, 0),
    },
    {
      name: 'for an unknown event',
      heat: live(99, 1),
    },
  ])('reports no position $name', ({ heat }) => {
    const current = currentHeatIndex(index, heat);

    expect(current).toBe(-1);
  });
});

describe('eventState', () => {
  it.each([
    {
      name: 'an event before the live heat as done',
      event: 0,
      live: live(2, 1),
      expected: EventState.Done,
    },
    {
      name: 'the event of the live heat as current',
      event: 1,
      live: live(2, 1),
      expected: EventState.Current,
    },
    {
      name: 'an event after the live heat as upcoming',
      event: 1,
      live: live(1, 1),
      expected: EventState.Upcoming,
    },
  ])('classifies $name', ({ event: position, live: heat, expected }) => {
    const current = currentHeatIndex(index, heat);

    const state = eventState(index, index.events[position]!, current);

    expect(state).toBe(expected);
  });

  it('returns null without a live heat', () => {
    const state = eventState(index, index.events[0]!, -1);

    expect(state).toBeNull();
  });

  it('never counts an event without heats as done', () => {
    const withEmpty = buildMeetIndex({
      ...meet,
      events: [
        event(3, {
          heatCount: 0,
        }),
        ...meet.events,
      ],
    });
    const current = currentHeatIndex(withEmpty, live(2, 1));

    const state = eventState(withEmpty, withEmpty.events[0]!, current);

    expect(state).toBe(EventState.Upcoming);
  });
});

describe('nextHeats', () => {
  it('returns the heats after the live heat', () => {
    const current = currentHeatIndex(index, live(1, 2));

    const heats = nextHeats(index, current, 3);

    expect(heats.map((heat) => heat.event.number)).toEqual([2]);
  });
});

describe('scheduledTime', () => {
  it('reads date and time as local time', () => {
    const scheduled = index.events[0]!;

    const time = scheduledTime(scheduled);

    expect(time).toEqual(new Date(2026, 4, 1, 10, 0));
  });

  it('returns null for an invalid date', () => {
    const scheduled = event(3, {
      date: '26/05/01',
    });

    const time = scheduledTime(scheduled);

    expect(time).toBeNull();
  });
});

describe('scheduleDelta', () => {
  it('interpolates the expected start of a heat between two events', () => {
    // heat 2 of 2 is expected halfway between 10:00 and 11:00
    const now = new Date(2026, 4, 1, 10, 40);

    const delta = scheduleDelta(index, 1, now);

    expect(delta).toBe(10 * 60_000);
  });

  it('uses the event start if the following event has no time', () => {
    const untimed = buildMeetIndex({
      ...meet,
      events: [
        meet.events[0]!,
        {
          ...meet.events[1]!,
          time: '',
        },
      ],
    });

    const delta = scheduleDelta(untimed, 1, new Date(2026, 4, 1, 10, 40));

    expect(delta).toBe(40 * 60_000);
  });

  it('uses the event start without a following event', () => {
    const now = new Date(2026, 4, 1, 10, 55);

    const delta = scheduleDelta(index, 2, now);

    expect(delta).toBe(-5 * 60_000);
  });

  it('returns null for an event without a time', () => {
    const untimed = buildMeetIndex({
      ...meet,
      events: [
        meet.events[0]!,
        {
          ...meet.events[1]!,
          time: '',
        },
      ],
    });

    const delta = scheduleDelta(untimed, 2, new Date(2026, 4, 1, 10, 40));

    expect(delta).toBeNull();
  });

  it('ignores a schedule of another day', () => {
    const now = new Date(2026, 4, 3, 10, 0);

    const delta = scheduleDelta(index, 1, now);

    expect(delta).toBeNull();
  });
});

describe('filterRows', () => {
  it('matches every word of the query in competitor or event', () => {
    const filters = {
      ...emptyFilters(),
      query: 'accented breaststroke',
    };

    const rows = filterRows(index, filters, noState);

    expect(rows.map((row) => row.entry.event)).toEqual([1]);
  });

  it('filters by club', () => {
    const filters = {
      ...emptyFilters(),
      clubs: ['Club One'],
    };

    const rows = filterRows(index, filters, noState);

    expect(rows).toHaveLength(1);
  });
});

describe('filterEvents', () => {
  it('matches every word of the query in competitor or event', () => {
    const filters = {
      ...emptyFilters(),
      query: 'accented breaststroke',
    };

    const events = filterEvents(index, filters, noState);

    expect(events.map((item) => item.number)).toEqual([1]);
  });

  it('finds an event by its phase', () => {
    const phased = buildMeetIndex({
      ...meet,
      events: [
        meet.events[0]!,
        {
          ...meet.events[1]!,
          phaseName: 'Final',
        },
      ],
    });
    const filters = {
      ...emptyFilters(),
      query: 'final',
    };

    const events = filterEvents(phased, filters, noState);

    expect(events.map((item) => item.number)).toEqual([2]);
  });

  it('finds an event by its own text', () => {
    const filters = {
      ...emptyFilters(),
      query: 'freestyle',
    };

    const events = filterEvents(index, filters, noState);

    expect(events.map((item) => item.number)).toEqual([2]);
  });

  it('filters by kind', () => {
    const filters = {
      ...emptyFilters(),
      kind: EventKind.Relay,
    };

    const events = filterEvents(index, filters, noState);

    expect(events.map((item) => item.number)).toEqual([2]);
  });

  it('filters by style', () => {
    const filters = {
      ...emptyFilters(),
      styles: ['Breaststroke'],
    };

    const events = filterEvents(index, filters, noState);

    expect(events.map((item) => item.number)).toEqual([1]);
  });

  it('hides done events when only upcoming ones are wanted', () => {
    const current = currentHeatIndex(index, live(2, 1));
    const stateOf = (item: EventResponse) => eventState(index, item, current);
    const filters = {
      ...emptyFilters(),
      upcomingOnly: true,
    };

    const events = filterEvents(index, filters, stateOf);

    expect(events.map((item) => item.number)).toEqual([2]);
  });
});

describe('filterClubs', () => {
  it('matches the query in the club name', () => {
    const filters = {
      ...emptyFilters(),
      query: 'two',
    };

    const clubs = filterClubs(index, filters);

    expect(clubs.map((club) => club.name)).toEqual(['Club Two']);
  });
});

describe('eventName', () => {
  it('adds the phase if it has a name', () => {
    const item = {
      name: '100 m Freestyle',
      phaseName: 'Final',
    };

    const name = eventName(item);

    expect(name).toBe('100 m Freestyle · Final');
  });

  it('returns the name of an event without a phase name', () => {
    const item = {
      name: '100 m Freestyle',
      phaseName: '',
    };

    const name = eventName(item);

    expect(name).toBe('100 m Freestyle');
  });
});
