import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import {
  apiFetch,
  setAuthenticationRequiredHandler,
  setRequestLanguage,
} from '@/api/client/client';
import { ApiError } from '@/api/client/errors';
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
        message: faker.lorem.sentence(),
      }),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).toHaveBeenCalledOnce();
  });

  it('skips the handler when the request asks for it', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(401, {
        code: 'authentication_required',
        message: faker.lorem.sentence(),
      }),
    );

    const error = await apiFetch(fakePath(), {
      skipAuthRedirect: true,
    }).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).not.toHaveBeenCalled();
  });

  it('skips the handler for wrong credentials', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(401, {
        code: 'invalid_credentials',
        message: faker.lorem.sentence(),
      }),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(onAuthenticationRequired).not.toHaveBeenCalled();
  });
});

describe('setRequestLanguage', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  afterEach(() => {
    setRequestLanguage(null);
  });

  it('sends the language with every request', async () => {
    const language = faker.location.countryCode().toLowerCase();
    setRequestLanguage(language);
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch(`/${faker.lorem.slug()}`);

    expect(lastRequest().init?.headers).toEqual({
      'Accept-Language': language,
    });
  });

  it('leaves the language to the browser without one', async () => {
    setRequestLanguage(null);
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch(`/${faker.lorem.slug()}`);

    expect(lastRequest().init?.headers).toEqual({});
  });
});

describe('apiFetch', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  it('requests the path below /api with the session cookie and returns the parsed body', async () => {
    const path = fakePath();
    const payload = {
      name: faker.lorem.word(),
    };
    fetchMock.mockResolvedValue(jsonResponse(200, payload));

    const body = await apiFetch(path);

    expect(body).toEqual(payload);
    expect(lastRequest().url).toBe(`/api${path}`);
    expect(lastRequest().init?.credentials).toBe('include');
  });

  it('returns undefined for an empty response', async () => {
    fetchMock.mockResolvedValue(jsonResponse(204, undefined));

    const body = await apiFetch(fakePath(), {
      method: 'DELETE',
    });

    expect(body).toBeUndefined();
  });

  it('returns the body as a blob when asked for one', async () => {
    const content = faker.lorem.sentence();
    fetchMock.mockResolvedValue(
      new Response(content, {
        status: 200,
      }),
    );

    const body = await apiFetch<Blob>(fakePath(), {
      responseType: 'blob',
    });

    expect(body).toBeInstanceOf(Blob);
    expect(await body.text()).toBe(content);
  });

  it('sends a plain object as JSON', async () => {
    const payload = {
      name: faker.lorem.word(),
    };
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch(fakePath(), {
      method: 'PUT',
      body: payload,
    });

    expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    expect(lastRequest().init?.headers).toEqual({
      'Content-Type': 'application/json',
    });
  });

  it('passes form data through without a JSON content type', async () => {
    fetchMock.mockResolvedValue(jsonResponse(204, undefined));
    const body = new FormData();
    body.append(faker.lorem.word(), faker.lorem.sentence());

    await apiFetch(fakePath(), {
      method: 'POST',
      body,
    });

    expect(lastRequest().init?.body).toBe(body);
    expect(lastRequest().init?.headers).toEqual({});
  });

  it('keeps the headers of the caller', async () => {
    const header = faker.string.alphanumeric(8);
    fetchMock.mockResolvedValue(jsonResponse(200, {}));

    await apiFetch(fakePath(), {
      method: 'PUT',
      body: {
        name: faker.lorem.word(),
      },
      headers: {
        'X-Test': header,
      },
    });

    expect(lastRequest().init?.headers).toEqual({
      'Content-Type': 'application/json',
      'X-Test': header,
    });
  });

  it('throws an ApiError with the status and code of a failed request', async () => {
    const message = faker.lorem.sentence();
    fetchMock.mockResolvedValue(
      jsonResponse(404, {
        code: 'not_found',
        message,
      }),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 404,
      code: 'not_found',
      message,
    });
  });

  it('reads the error of a failed blob request as JSON', async () => {
    const message = faker.lorem.sentence();
    fetchMock.mockResolvedValue(
      jsonResponse(404, {
        code: 'not_found',
        message,
      }),
    );

    const error = await apiFetch(fakePath(), {
      responseType: 'blob',
    }).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 404,
      code: 'not_found',
      message,
    });
  });

  it('carries the field errors of a validation failure', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(422, {
        code: 'validation_failed',
        message: faker.lorem.sentence(),
        errors: {
          name: [
            {
              tag: 'required',
            },
          ],
        },
      }),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

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
    const requestId = faker.string.uuid();
    const retryAfter = faker.number.int({
      min: 1,
      max: 120,
    });
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 'rate_limited',
          message: faker.lorem.sentence(),
          requestId,
        }),
        {
          status: 429,
          headers: {
            'Retry-After': String(retryAfter),
          },
        },
      ),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toMatchObject({
      retryAfter,
      requestId,
    });
  });

  it('ignores a Retry-After header that is not a number of seconds', async () => {
    const requestId = faker.string.uuid();
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 'rate_limited',
          message: faker.lorem.sentence(),
        }),
        {
          status: 429,
          headers: {
            'Retry-After': faker.date.future().toUTCString(),
            'X-Request-ID': requestId,
          },
        },
      ),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toMatchObject({
      retryAfter: undefined,
      requestId,
    });
  });

  it('reports a server error when the error body is not JSON', async () => {
    fetchMock.mockResolvedValue(
      new Response(faker.lorem.sentence(), {
        status: 502,
      }),
    );

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toMatchObject({
      status: 502,
      code: 'server_error',
    });
  });

  it('reports a network error when no answer arrives', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      status: 0,
      code: 'network',
    });
  });

  it('reports a network error when the body stops arriving', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      text: async () => Promise.reject(new TypeError('network error')),
      json: async () => Promise.reject(new TypeError('network error')),
    } as unknown as Response);

    const error = await apiFetch(fakePath()).catch((err: unknown) => err);

    expect(error).toMatchObject({
      status: 0,
      code: 'network',
    });
  });

  it('reports a timeout when the request takes too long', async () => {
    vi.useFakeTimers();
    fetchMock.mockImplementation(
      async (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('aborted', 'AbortError')),
          );
        }),
    );

    const request = apiFetch(fakePath()).catch((err: unknown) => err);
    await vi.advanceTimersByTimeAsync(30_000);

    expect(await request).toMatchObject({
      status: 0,
      code: 'timeout',
    });
  });

  it('waits for the timeout of the call instead of the default', async () => {
    vi.useFakeTimers();
    fetchMock.mockImplementation(
      async (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('aborted', 'AbortError')),
          );
        }),
    );
    let settled = false;

    const request = apiFetch(fakePath(), {
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
    fetchMock.mockImplementation(async (_url, init) =>
      Promise.resolve({
        ok: true,
        text: async () =>
          new Promise<string>((_resolve, reject) => {
            init?.signal?.addEventListener('abort', () =>
              reject(new DOMException('aborted', 'AbortError')),
            );
          }),
      } as Response),
    );

    const request = apiFetch(fakePath()).catch((err: unknown) => err);
    await vi.advanceTimersByTimeAsync(30_000);

    expect(await request).toMatchObject({
      status: 0,
      code: 'timeout',
    });
  });

  it('passes the cancel of the caller through instead of reporting a timeout', async () => {
    fetchMock.mockImplementation(
      async (_url, init) =>
        new Promise((_resolve, reject) => {
          init?.signal?.addEventListener('abort', () =>
            reject(new DOMException('aborted', 'AbortError')),
          );
        }),
    );
    const controller = new AbortController();

    const request = apiFetch(fakePath(), {
      signal: controller.signal,
    }).catch((err: unknown) => err);
    controller.abort();

    const error = await request;
    expect(error).toBeInstanceOf(DOMException);
    expect((error as DOMException).name).toBe('AbortError');
  });
});

/** Returns a path of the API that no test gives a meaning. */
function fakePath(): string {
  return `/${faker.lorem.slug()}/${faker.string.uuid()}`;
}
