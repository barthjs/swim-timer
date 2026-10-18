import { beforeEach, describe, expect, it, vi } from 'vitest';

import { apiFetch, setAuthenticationRequiredHandler } from '@/api/client';
import { ApiError } from '@/api/errors';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('setAuthenticationRequiredHandler', () => {
  const { fetchMock } = useFetchMock();
  const onAuthenticationRequired = vi.fn<() => void>();

  beforeEach(() => {
    onAuthenticationRequired.mockReset();
    setAuthenticationRequiredHandler(onAuthenticationRequired);
  });

  it('calls the handler for a missing or expired session', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(401, {
        code: 'authentication_required',
        message: 'login',
      }),
    );

    const error = await apiFetch('/settings').catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).toHaveBeenCalledOnce();
  });

  it('skips the handler when the request asks for it', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(401, {
        code: 'authentication_required',
        message: 'login',
      }),
    );

    const error = await apiFetch('/auth/me', {
      skipAuthRedirect: true,
    }).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).not.toHaveBeenCalled();
  });

  it('skips the handler for wrong credentials', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(401, {
        code: 'invalid_credentials',
        message: 'wrong',
      }),
    );

    const error = await apiFetch('/auth/login').catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).not.toHaveBeenCalled();
  });

  it('skips the handler for other errors', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(403, {
        code: 'forbidden',
        message: 'no',
      }),
    );

    const error = await apiFetch('/settings').catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).not.toHaveBeenCalled();
  });
});

describe('apiFetch', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  it('returns the parsed response body', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(200, {
        saved: true,
      }),
    );

    const body = await apiFetch('/items/a');

    expect(body).toEqual({
      saved: true,
    });
  });

  it('returns undefined for an empty response', async () => {
    fetchMock.mockResolvedValue(jsonResponse(204, undefined));

    const body = await apiFetch('/items/a', {
      method: 'DELETE',
    });

    expect(body).toBeUndefined();
  });

  it('prefixes the path with /api', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch('/items');

    expect(lastRequest().url).toBe('/api/items');
  });

  it('sends the session cookie', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch('/items');

    expect(lastRequest().init?.credentials).toBe('include');
  });

  it('sends a plain object as JSON', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch('/items/a', {
      method: 'PUT',
      body: {
        name: 'a',
      },
    });

    expect(lastRequest().init?.body).toBe('{"name":"a"}');
    expect(lastRequest().init?.headers).toEqual({
      'Content-Type': 'application/json',
    });
  });

  it('passes form data through without a JSON content type', async () => {
    fetchMock.mockResolvedValue(jsonResponse(204, undefined));
    const body = new FormData();
    body.append('file', 'content');

    await apiFetch('/uploads', {
      method: 'POST',
      body,
    });

    expect(lastRequest().init?.body).toBe(body);
    expect(lastRequest().init?.headers).toEqual({});
  });

  it('keeps the headers of the caller', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch('/items/a', {
      method: 'PUT',
      body: {
        name: 'a',
      },
      headers: {
        'X-Test': '1',
      },
    });

    expect(lastRequest().init?.headers).toEqual({
      'Content-Type': 'application/json',
      'X-Test': '1',
    });
  });

  it('throws an ApiError with the status and code of a failed request', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(404, {
        code: 'not_found',
        message: 'missing',
      }),
    );

    const error = await apiFetch('/items/a').catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 404,
      code: 'not_found',
      message: 'missing',
    });
  });

  it('carries the field errors of a validation failure', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(422, {
        code: 'validation_failed',
        message: 'invalid',
        errors: {
          name: [
            {
              tag: 'required',
            },
          ],
        },
      }),
    );

    const error = await apiFetch('/items/a').catch((err: unknown) => err);

    expect(error).toMatchObject({
      fields: {
        name: [
          {
            tag: 'required',
          },
        ],
      },
    });
  });

  it('carries the wait and the request id of a rate-limited request', async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 'rate_limited',
          message: 'slow down',
          requestId: 'req-1',
        }),
        {
          status: 429,
          headers: {
            'Retry-After': '30',
          },
        },
      ),
    );

    const error = await apiFetch('/auth/login').catch((err: unknown) => err);

    expect(error).toMatchObject({
      retryAfter: 30,
      requestId: 'req-1',
    });
  });

  it('ignores a Retry-After header that is not a number of seconds', async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 'rate_limited',
          message: 'slow down',
        }),
        {
          status: 429,
          headers: {
            'Retry-After': 'Wed, 21 Oct 2026 07:28:00 GMT',
            'X-Request-ID': 'req-2',
          },
        },
      ),
    );

    const error = await apiFetch('/auth/login').catch((err: unknown) => err);

    expect(error).toMatchObject({
      retryAfter: undefined,
      requestId: 'req-2',
    });
  });

  it('reports a server error when the error body is not JSON', async () => {
    fetchMock.mockResolvedValue(
      new Response('<html>Bad Gateway</html>', {
        status: 502,
      }),
    );

    const error = await apiFetch('/meta').catch((err: unknown) => err);

    expect(error).toMatchObject({
      status: 502,
      code: 'server_error',
    });
  });

  it('rethrows network errors', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));

    const error = await apiFetch('/meta').catch((err: unknown) => err);

    expect(error).toBeInstanceOf(TypeError);
  });

  it('reports a timeout when the request takes too long', async () => {
    vi.useFakeTimers();
    fetchMock.mockImplementation(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('aborted', 'AbortError')),
          );
        }),
    );

    const request = apiFetch('/meta').catch((err: unknown) => err);
    await vi.advanceTimersByTimeAsync(30_000);

    expect(await request).toMatchObject({
      status: 0,
      code: 'timeout',
    });
  });

  it('waits for the timeout of the call instead of the default', async () => {
    vi.useFakeTimers();
    fetchMock.mockImplementation(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('aborted', 'AbortError')),
          );
        }),
    );
    let settled = false;

    const request = apiFetch('/meta', {
      timeout: 60_000,
    })
      .catch((err: unknown) => err)
      .finally(() => {
        settled = true;
      });
    await vi.advanceTimersByTimeAsync(30_000);
    expect(settled).toBe(false);
    await vi.advanceTimersByTimeAsync(30_000);

    expect(await request).toMatchObject({
      status: 0,
      code: 'timeout',
    });
  });

  it('reports a timeout when the body stops arriving', async () => {
    vi.useFakeTimers();
    fetchMock.mockImplementation((_url, init) =>
      Promise.resolve({
        ok: true,
        text: () =>
          new Promise<string>((_resolve, reject) => {
            init?.signal?.addEventListener('abort', () =>
              reject(new DOMException('aborted', 'AbortError')),
            );
          }),
      } as Response),
    );

    const request = apiFetch('/meta').catch((err: unknown) => err);
    await vi.advanceTimersByTimeAsync(30_000);

    expect(await request).toMatchObject({
      status: 0,
      code: 'timeout',
    });
  });

  it('passes the cancel of the caller through instead of reporting a timeout', async () => {
    fetchMock.mockImplementation(
      (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('aborted', 'AbortError')),
          );
        }),
    );
    const controller = new AbortController();

    const request = apiFetch('/meta', {
      signal: controller.signal,
    }).catch((err: unknown) => err);
    controller.abort();

    const error = await request;
    expect(error).toBeInstanceOf(DOMException);
    expect((error as DOMException).name).toBe('AbortError');
  });
});
