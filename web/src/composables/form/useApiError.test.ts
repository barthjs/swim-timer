import { describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import { ApiError } from '@/api/client/errors';
import { useApiError } from '@/composables/form/useApiError';

const messages: Record<string, string> = {
  'errors.invalid_credentials': faker.lorem.sentence(),
  'errors.server_error': faker.lorem.sentence(),
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
        message: faker.lorem.sentence(),
        status: 401,
        code: 'invalid_credentials',
      });

      const message = messageFor(error);

      expect(message).toBe(messages['errors.invalid_credentials']);
    });

    it('names the wait of a rate-limited request in seconds', () => {
      const { messageFor } = useApiError();
      const error = new ApiError({
        message: faker.lorem.sentence(),
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
        message: faker.lorem.sentence(),
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
        message: faker.lorem.sentence(),
        status: 418,
        code: faker.lorem.word(),
      });

      const message = messageFor(error);

      expect(message).toBe(messages['errors.server_error']);
    });

    it.each([
      {
        kind: 'an error',
        error: new Error(faker.lorem.sentence()),
      },
      {
        kind: 'a text',
        error: faker.lorem.sentence(),
      },
    ])('falls back to the generic message for $kind that is no ApiError', ({ error }) => {
      const { messageFor } = useApiError();

      const message = messageFor(error);

      expect(message).toBe(messages['errors.server_error']);
    });
  });
});
