import type { Lane } from '@/api/types';

/** HeatRun identifies a run of a heat: the heat by its event and number, and the run by its start time. */
export type HeatRun = {
  heat: string;
  startTime: string;
};

/** SplitTracker decides which lane times are new and should be shown by the splits overlay. */
export type SplitTracker = {
  // update takes the timed lanes of a run of a heat and returns the lanes whose time is new.
  update: (run: HeatRun, lanes: Lane[]) => Lane[];
};

/**
 * heatRun identifies a run of a heat. A new start of the same heat, such as after a false start,
 * gets a new start time and therefore another run.
 */
export function heatRun(event: number, heat: number, startTime: string | undefined): HeatRun {
  return {
    heat: `${event}-${heat}`,
    startTime: startTime ?? '',
  };
}

/**
 * createSplitTracker remembers the last time seen per lane. The first update and the first update of another heat
 * only record the times the heat already has, so neither an overlay opened during a heat nor a switch back
 * to an earlier heat shows every earlier time at once. A new run of the same heat shows its times, since its start
 * can arrive together with its first time, for example when a held back start is confirmed by that time.
 * Every new run forgets the recorded times, so a time equal to one of the run before is shown.
 */
export function createSplitTracker(): SplitTracker {
  const times = new Map<number, number>();
  let current: HeatRun | null = null;

  /** update records the times of the lanes and returns the lanes whose time changed. */
  function update(run: HeatRun, lanes: Lane[]): Lane[] {
    const sameHeat = current?.heat === run.heat;
    if (!sameHeat || current?.startTime !== run.startTime) {
      times.clear();
      current = run;
    }

    const changed: Lane[] = [];
    for (const lane of lanes) {
      if (times.get(lane.number) === lane.time) continue;
      times.set(lane.number, lane.time);
      if (sameHeat) changed.push(lane);
    }
    return changed;
  }

  return {
    update,
  };
}
