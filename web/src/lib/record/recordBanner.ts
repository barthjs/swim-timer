import type { Lane, LaneRecord, LiveHeat } from '@/api/types/live';

/** A lane that broke or equalled a record, as the info overlay announces it. */
export type RecordBanner = {
  // Identifies the lane within the meet, so every lane is announced once
  key: string;
  // Identifies the heat of the record, see recordHeat
  heat: string;
  lane: Lane;
  record: LaneRecord;
};

/**
 * Returns the lanes of the live heat with a record by rank, each with its first broken or else first equalled record.
 */
export function recordBanners(state: LiveHeat | null): RecordBanner[] {
  if (!state) return [];

  const lanes = Object.values(state.lanes)
    .filter((lane) => (lane.records?.length ?? 0) > 0)
    .sort((a, b) => a.rank - b.rank || a.number - b.number);

  const heat = recordHeat(state);
  const banners: RecordBanner[] = [];
  for (const lane of lanes) {
    const records = lane.records ?? [];
    const record = records.find((item) => !item.equalled) ?? records[0];
    if (!record) continue;
    banners.push({
      key: `${heat}-${lane.number}`,
      heat,
      lane,
      record,
    });
  }
  return banners;
}

/** Identifies the heat of the live heat, empty without one. */
export function recordHeat(state: Pick<LiveHeat, 'event' | 'heat'> | null): string {
  return state ? `${state.event}-${state.heat}` : '';
}
