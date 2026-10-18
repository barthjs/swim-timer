import { describe, expect, it, vi } from 'vitest';

import { ApiError } from '@/api/errors';
import { useApiError } from '@/composables/useApiError';

const messages: Record<string, string> = {
  'errors.invalid_credentials': 'Wrong password',
  'errors.server_error': 'Something went wrong',
};

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: { count: number }) =>
      values ? `${key}:${values.count}` : (messages[key] ?? key),
    te: (key: string) => key in messages,
  }),
}));

describe('useApiError', () => {
  describe('messageFor', () => {
    it('returns the localized message of a known error code', () => {
      const { messageFor } = useApiError();
      const error = new ApiError({
        message: 'raw',
        status: 401,
        code: 'invalid_credentials',
      });

      const message = messageFor(error);

      expect(message).toBe('Wrong password');
    });

    it('names the wait of a rate-limited request in seconds', () => {
      const { messageFor } = useApiError();
      const error = new ApiError({
        message: 'raw',
        status: 429,
        code: 'rate_limited',
        retryAfter: 42,
      });

      const message = messageFor(error);

      expect(message).toBe('errors.rate_limited_seconds:42');
    });

    it('rounds longer waits up to whole minutes', () => {
      const { messageFor } = useApiError();
      const error = new ApiError({
        message: 'raw',
        status: 429,
        code: 'rate_limited',
        retryAfter: 61,
      });

      const message = messageFor(error);

      expect(message).toBe('errors.rate_limited_minutes:2');
    });

    it('falls back to the generic message for an unknown code', () => {
      const { messageFor } = useApiError();
      const error = new ApiError({
        message: 'raw',
        status: 418,
        code: 'teapot',
      });

      const message = messageFor(error);

      expect(message).toBe('Something went wrong');
    });

    it.each([
      {
        kind: 'an error',
        error: new Error('network'),
      },
      {
        kind: 'a text',
        error: 'text',
      },
    ])('falls back to the generic message for $kind that is no ApiError', ({ error }) => {
      const { messageFor } = useApiError();

      const message = messageFor(error);

      expect(message).toBe('Something went wrong');
    });
  });
});
