import { describe, expect, it } from 'vitest';

import { ApiError } from '@/api/errors';

describe('ApiError', () => {
  it('is an Error that carries the details of the response', () => {
    const error = new ApiError({
      message: 'invalid',
      status: 422,
      code: 'validation_failed',
      fields: {
        name: [
          {
            tag: 'required',
          },
        ],
      },
      retryAfter: 5,
      requestId: 'abc',
    });

    expect(error).toBeInstanceOf(Error);
    expect(error.message).toBe('invalid');
    expect(error).toMatchObject({
      status: 422,
      code: 'validation_failed',
      fields: {
        name: [
          {
            tag: 'required',
          },
        ],
      },
      retryAfter: 5,
      requestId: 'abc',
    });
  });

  it('leaves the optional details undefined', () => {
    const error = new ApiError({
      message: 'missing',
      status: 404,
      code: 'not_found',
    });

    expect(error.fields).toBeUndefined();
    expect(error.retryAfter).toBeUndefined();
    expect(error.requestId).toBeUndefined();
  });
});
