import type { ApiErrorResponse, ValidationErrorResponse } from '@/types/api';
import {
  ApiError,
  AuthenticationError,
  BadRequestError,
  NotFoundError,
  ServerError,
  UnauthorizedError,
  ValidationError,
} from '@/types/api';

export async function apiClient<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);

  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(buildUrl(endpoint), {
    ...options,
    headers,
  });

  if (!response.ok) {
    await handleErrorResponse(response, endpoint);
  }

  const contentType = response.headers.get('content-type');

  if (contentType?.includes('application/json')) {
    return await response.json();
  }

  if (
    contentType?.includes('application/pdf') ||
    contentType?.includes('image/') ||
    contentType?.includes('application/octet-stream')
  ) {
    return (await response.blob()) as T;
  }

  return (await response.text()) as T;
}

export async function authApiClient<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = sessionStorage.getItem('secretKey');
  const headers = new Headers(options.headers);
  headers.set('Authorization', token ? `Bearer ${token}` : '');

  return apiClient<T>(endpoint, {
    ...options,
    headers,
  });
}

async function handleErrorResponse(response: Response, endpoint: string = ''): Promise<never> {
  const contentType = response.headers.get('content-type');

  if (contentType?.includes('application/json')) {
    const data = (await response.json()) as ApiErrorResponse | ValidationErrorResponse;

    switch (response.status) {
      case 400:
        throw new BadRequestError(data.message);
      case 401:
        if (endpoint.includes('/auth/login')) {
          throw new AuthenticationError(data.message);
        }
        throw new UnauthorizedError(data.message);
      case 404:
        throw new NotFoundError(data.message);
      case 422:
        if ('errors' in data && data.errors) {
          throw new ValidationError(data.errors, data.message);
        }
        throw new ApiError(response.status, data.message);
      case 500:
      case 502:
      case 503:
      case 504:
        throw new ServerError(data.message);
      default:
        throw new ApiError(response.status, data.message);
    }
  }

  throw new ApiError(response.status, `${response.status} ${response.statusText}`);
}

function buildUrl(endpoint: string): string {
  return `/api/${endpoint.replace(/^\/+/, '')}`;
}
