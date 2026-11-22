import { ApiError, type ApiErrorResponse } from '@/api/client/errors';

const baseURL = '/api';
const REQUEST_TIMEOUT_MS = 30_000;

/** Matches the time the server allows an upload route. */
export const UPLOAD_TIMEOUT_MS = 5 * 60_000;

// Runs when a request fails because the session is missing or expired.
// It is registered by the app instead of imported here, so another entry point can use the client
// without pulling in the router, its pages, and Nuxt UI.
let onAuthenticationRequired: (() => void) | null = null;

/**
 * Registers what happens when the session is missing or expired.
 */
export function setAuthenticationRequiredHandler(handler: () => void): void {
  onAuthenticationRequired = handler;
}

// The language of the interface, sent with every request so the server answers in it.
// Without one the browser sends its own languages.
let requestLanguage: string | null = null;

/** Sets the language of the interface that every request sends to the server. */
export function setRequestLanguage(language: string | null): void {
  requestLanguage = language;
}

export type ApiFetchOptions = Omit<RequestInit, 'body' | 'headers'> & {
  body?: BodyInit | Record<string, unknown> | null;
  headers?: Record<string, string>;
  skipAuthRedirect?: boolean;
  // Milliseconds until the request is aborted, the default fits every route except uploads
  timeout?: number;
  // How the body of a successful response is read, JSON by default and a blob for a file to download.
  // A failed response is always read as JSON.
  responseType?: 'json' | 'blob';
};

/** Reads and parses a response body. */
async function parseJSON<T>(response: Response): Promise<T | undefined> {
  const text = await response.text();
  if (!text) return undefined;

  try {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the API answers in the shape its caller declares
    return JSON.parse(text) as T;
  } catch {
    return undefined;
  }
}

/** Sends a request to the API and returns the parsed response, or the body as a blob, and throws an ApiError. */
export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const {
    skipAuthRedirect,
    timeout: timeoutMs,
    responseType,
    body,
    headers,
    signal,
    ...rest
  } = options;

  const isPlainBody = body !== undefined && body !== null && !(body instanceof FormData);

  // The timeout covers reading the body as well, a response that stops halfway would hang otherwise.
  // A signal of the caller still cancels the request.
  const timeoutController = new AbortController();
  const timeout = setTimeout(() => timeoutController.abort(), timeoutMs ?? REQUEST_TIMEOUT_MS);
  const requestSignal = signal
    ? AbortSignal.any([timeoutController.signal, signal])
    : timeoutController.signal;

  try {
    return await send<T>(
      `${baseURL}${path}`,
      {
        ...rest,
        headers: {
          ...(requestLanguage
            ? {
                'Accept-Language': requestLanguage,
              }
            : {}),
          ...(isPlainBody
            ? {
                'Content-Type': 'application/json',
              }
            : {}),
          ...headers,
        },
        credentials: 'include',
        body: isPlainBody ? JSON.stringify(body) : (body as BodyInit | null | undefined),
        signal: requestSignal,
      },
      skipAuthRedirect,
      responseType,
    );
  } catch (err) {
    if (
      timeoutController.signal.aborted &&
      err instanceof DOMException &&
      err.name === 'AbortError'
    ) {
      throw new ApiError({
        message: 'request timed out',
        status: 0,
        code: 'timeout',
      });
    }
    // fetch and the reading of the body reject with a TypeError only when the answer does not arrive,
    // which is an error of the API like any other.
    if (err instanceof TypeError) {
      throw new ApiError({
        message: 'network error',
        status: 0,
        code: 'network',
      });
    }
    throw err;
  } finally {
    clearTimeout(timeout);
  }
}

/** Runs the request, reads its body, and turns a failed response into an ApiError. */
async function send<T>(
  url: string,
  init: RequestInit,
  skipAuthRedirect?: boolean,
  responseType?: 'json' | 'blob',
): Promise<T> {
  const response = await fetch(url, init);

  if (!response.ok) {
    const data = (await parseJSON<ApiErrorResponse>(response)) ?? {
      code: 'server_error',
      message: 'request failed',
    };

    // Only a missing or expired session leads to the login. A wrong password (invalid_credentials) is also a 401,
    // but is an expected outcome on the login form itself.
    if (response.status === 401 && data.code === 'authentication_required' && !skipAuthRedirect) {
      onAuthenticationRequired?.();
    }

    throw new ApiError({
      message: data.message,
      status: response.status,
      code: data.code,
      fields: data.errors,
      retryAfter: parseRetryAfter(response.headers.get('Retry-After')),
      requestId: data.requestId ?? response.headers.get('X-Request-ID') ?? undefined,
    });
  }

  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- a caller that asks for a blob declares it as the result
  if (responseType === 'blob') return (await response.blob()) as T;
  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- a caller reads no body of an answer without content
  return (await parseJSON<T>(response)) as T;
}

/**
 * Reads the Retry-After header in its seconds form, the only form the server sends.
 */
function parseRetryAfter(value: string | null): number | undefined {
  if (value === null) return undefined;
  const seconds = Number(value);
  return Number.isInteger(seconds) && seconds >= 0 ? seconds : undefined;
}
