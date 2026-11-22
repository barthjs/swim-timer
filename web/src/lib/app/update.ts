import { ApiError, isGatewayError } from '@/api/client/errors';

/** How long the restart after an update may take before it counts as failed. */
export const RESTART_TIMEOUT_MS = 3 * 60 * 1000;

export const RestartState = {
  Waiting: 'waiting',
  Restarted: 'restarted',
  TimedOut: 'timedOut',
} as const;
export type RestartState = (typeof RestartState)[keyof typeof RestartState];

/**
 * Reports whether a request failed because the server is not available, as while it restarts.
 * A request without an answer, a request that timed out, and a gateway error of a reverse proxy count,
 * any other HTTP error response does not.
 */
export function isServerUnavailable(err: unknown): boolean {
  return err instanceof ApiError && (err.status === 0 || isGatewayError(err.status));
}

/**
 * Reports whether a failed status request during an update means that the server restarts.
 * Besides an unavailable server this is a missing session, since the sessions live in memory
 * and the restarted server no longer knows the one of the running update.
 */
export function isRestartAnswer(err: unknown): boolean {
  if (err instanceof ApiError && err.status === 401) return true;
  return isServerUnavailable(err);
}

/**
 * Tells whether the server is back after an update with any version other than the one before,
 * since the update installs the latest release, which can be newer than the one the check showed.
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
