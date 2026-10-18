import { describe, expect, it } from 'vitest';

import { ApiError } from '@/api/errors';
import {
  RESTART_TIMEOUT_MS,
  RestartState,
  isRestartAnswer,
  isServerUnavailable,
  restartState,
} from '@/lib/update';

describe('isServerUnavailable', () => {
  it.each([
    {
      name: 'a failed fetch',
      error: new TypeError('Failed to fetch'),
    },
    {
      name: 'a timeout',
      error: new ApiError({
        message: 'request timed out',
        status: 0,
        code: 'timeout',
      }),
    },
  ])('counts $name as an unavailable server', ({ error }) => {
    const unavailable = isServerUnavailable(error);

    expect(unavailable).toBe(true);
  });

  it.each([502, 503, 504])(
    'counts the gateway error %i of a reverse proxy as an unavailable server',
    (status) => {
      const error = new ApiError({
        message: 'bad gateway',
        status,
        code: 'server_error',
      });

      const unavailable = isServerUnavailable(error);

      expect(unavailable).toBe(true);
    },
  );

  it.each([
    {
      name: 'another HTTP error response',
      error: new ApiError({
        message: 'request failed',
        status: 500,
        code: 'server_error',
      }),
    },
    {
      name: 'another error',
      error: new Error('other'),
    },
  ])('does not count $name as an unavailable server', ({ error }) => {
    const unavailable = isServerUnavailable(error);

    expect(unavailable).toBe(false);
  });
});

describe('isRestartAnswer', () => {
  it.each([
    {
      name: 'a failed fetch',
      error: new TypeError('Failed to fetch'),
    },
    {
      name: 'a gateway error',
      error: new ApiError({
        message: 'bad gateway',
        status: 502,
        code: 'server_error',
      }),
    },
  ])('counts $name of an unavailable server as a restart', ({ error }) => {
    const restarted = isRestartAnswer(error);

    expect(restarted).toBe(true);
  });

  it('counts a missing session as a restart, since the restarted server forgot it', () => {
    const error = new ApiError({
      message: 'authentication required',
      status: 401,
      code: 'authentication_required',
    });

    const restarted = isRestartAnswer(error);

    expect(restarted).toBe(true);
  });

  it.each([
    {
      name: 'a server error',
      error: new ApiError({
        message: 'request failed',
        status: 500,
        code: 'server_error',
      }),
    },
    {
      name: 'a forbidden answer',
      error: new ApiError({
        message: 'forbidden',
        status: 403,
        code: 'forbidden',
      }),
    },
  ])('does not count $name as a restart', ({ error }) => {
    const restarted = isRestartAnswer(error);

    expect(restarted).toBe(false);
  });
});

describe('restartState', () => {
  const startedAt = 1_000;

  it.each([
    {
      name: 'does not answer',
      version: null,
    },
    {
      name: 'reports the previous version',
      version: '1.0.0',
    },
  ])('waits while the server $name', ({ version }) => {
    const state = restartState('1.0.0', version, startedAt, startedAt + 1);

    expect(state).toBe(RestartState.Waiting);
  });

  it.each([
    {
      name: 'before',
      now: startedAt + 1,
    },
    {
      name: 'at',
      now: startedAt + RESTART_TIMEOUT_MS,
    },
  ])(
    'counts any other version $name the timeout as restarted, also one newer than the checked release',
    ({ now }) => {
      const state = restartState('1.0.0', '1.2.0', startedAt, now);

      expect(state).toBe(RestartState.Restarted);
    },
  );

  it.each([
    {
      name: 'without an answer',
      version: null,
    },
    {
      name: 'with the previous version',
      version: '1.0.0',
    },
  ])(
    'times out when the server does not come back with another version in time, $name',
    ({ version }) => {
      const state = restartState('1.0.0', version, startedAt, startedAt + RESTART_TIMEOUT_MS);

      expect(state).toBe(RestartState.TimedOut);
    },
  );
});
