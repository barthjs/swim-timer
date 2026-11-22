import { Phase, type Lane, type LiveHeat } from '@/api/types/live';

/** Reports whether a lane has a rank, since 0 stands for no rank yet. */
export function hasRank(lane: Pick<Lane, 'rank'>): boolean {
  return lane.rank > 0;
}

/** Reports whether a lane is listed with its rank in the results, which needs a finish and a rank. */
export function isResultRanked(lane: Pick<Lane, 'rank' | 'finished'>): boolean {
  return lane.finished && hasRank(lane);
}

/**
 * Orders lanes as in the official results: lanes with a finish and a rank by rank,
 * then every other lane by lane number. Equal ranks keep the order of the lane numbers.
 */
export function compareResults(
  a: Pick<Lane, 'number' | 'rank' | 'finished'>,
  b: Pick<Lane, 'number' | 'rank' | 'finished'>,
): number {
  const aRanked = isResultRanked(a);
  const bRanked = isResultRanked(b);
  if (aRanked !== bRanked) {
    return aRanked ? -1 : 1;
  }
  if (aRanked && a.rank !== b.rank) {
    return a.rank - b.rank;
  }
  return a.number - b.number;
}

/** Returns the name of the phase if the heat belongs to a final, otherwise an empty string. */
export function finalPhaseName(heat: Pick<LiveHeat, 'phase' | 'phaseName'>): string {
  if (heat.phase !== Phase.Final) return '';
  return heat.phaseName.trim();
}

/**
 * Reports whether a heat number can be shown together with the heat count of its event, which a heat beyond it cannot.
 */
export function showsHeatCount(heat: number, heatCount: number): boolean {
  return heat >= 1 && heat <= heatCount;
}
