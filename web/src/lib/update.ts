import { ApiError } from '@/api/errors';

/** How long the restart after an update may take before it counts as failed. */
export const RESTART_TIMEOUT_MS = 3 * 60 * 1000;

export const RestartState = {
  Waiting: 'waiting',
  Restarted: 'restarted',
  TimedOut: 'timedOut',
} as const;
export type RestartState = (typeof RestartState)[keyof typeof RestartState];

/** HTTP status codes a reverse proxy answers with while the server behind it does not answer. */
const GATEWAY_STATUSES: readonly number[] = [502, 503, 504];

/**
 * isServerUnavailable reports whether a request failed because the server is not available, as while it restarts.
 * A request without an answer, a request that timed out, and a gateway error of a reverse proxy count,
 * any other HTTP error response does not.
 */
export function isServerUnavailable(err: unknown): boolean {
  if (err instanceof ApiError) return err.status === 0 || GATEWAY_STATUSES.includes(err.status);
  return err instanceof TypeError;
}

/**
 * isRestartAnswer reports whether a failed status request during an update means that the server restarts.
 * Besides an unavailable server this is a missing session, since the sessions live in memory
 * and the restarted server no longer knows the one of the running update.
 */
export function isRestartAnswer(err: unknown): boolean {
  if (err instanceof ApiError && err.status === 401) return true;
  return isServerUnavailable(err);
}

/**
 * restartState tells whether the server is back after an update. The update installs the latest release
 * at the time it runs, which can be newer than the one the check showed, so any version other than the one
 * before the update counts as restarted. reported is null while the server does not answer.
 */
export function restartState(
  previousVersion: string,
  reported: string | null,
  startedAt: number,
  now: number,
): RestartState {
  if (reported !== null && reported !== previousVersion) return RestartState.Restarted;
  if (now - startedAt >= RESTART_TIMEOUT_MS) return RestartState.TimedOut;
  return RestartState.Waiting;
}
