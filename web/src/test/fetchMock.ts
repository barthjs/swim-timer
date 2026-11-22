import { afterEach, beforeEach, vi } from 'vitest';

/** Builds a response with a JSON body, or without a body for undefined. */
export function jsonResponse(status: number, body: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
  });
}

/**
 * Replaces the global fetch for every test of the calling file and restores it afterward.
 */
export function useFetchMock() {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  /** Returns the URL and options of the most recent call. */
  function lastRequest(): { url: unknown; init: RequestInit | undefined } {
    const call = fetchMock.mock.calls.at(-1);
    return {
      url: call?.[0],
      init: call?.[1],
    };
  }

  return {
    fetchMock,
    lastRequest,
  };
}
