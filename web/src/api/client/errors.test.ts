import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { ApiError, fieldErrorOf, isGatewayError } from '@/api/client/errors';

describe('ApiError', () => {
  it('is an Error that carries the details of the response', () => {
    const message = faker.lorem.sentence();
    const retryAfter = faker.number.int({
      min: 1,
      max: 120,
    });
    const requestId = faker.string.uuid();

    const error = new ApiError({
      message,
      status: 422,
      code: 'validation_failed',
      fields: {
        name: [
          {
            tag: 'required',
          },
        ],
      },
      retryAfter,
      requestId,
    });

    expect(error).toBeInstanceOf(Error);
    expect(error.message).toBe(message);
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
      retryAfter,
      requestId,
    });
  });

  it('leaves the optional details undefined', () => {
    const error = new ApiError({
      message: faker.lorem.sentence(),
      status: 404,
      code: 'not_found',
    });

    expect(error.fields).toBeUndefined();
    expect(error.retryAfter).toBeUndefined();
    expect(error.requestId).toBeUndefined();
  });
});

describe('fieldErrorOf', () => {
  it('returns the first error of the field', () => {
    const field = faker.lorem.word();
    const param = faker.string.numeric(2);
    const error = new ApiError({
      message: faker.lorem.sentence(),
      status: 422,
      code: 'validation_failed',
      fields: {
        [field]: [
          {
            tag: 'max',
            param,
          },
          {
            tag: 'required',
          },
        ],
      },
    });

    const fieldError = fieldErrorOf(error, field);

    expect(fieldError).toEqual({
      tag: 'max',
      param,
    });
  });

  it('returns undefined without an error of the field', () => {
    const error = new ApiError({
      message: faker.lorem.sentence(),
      status: 422,
      code: 'validation_failed',
      fields: {
        name: [
          {
            tag: 'required',
          },
        ],
      },
    });

    const fieldError = fieldErrorOf(error, 'email');

    expect(fieldError).toBeUndefined();
  });

  it('returns undefined for an error that did not come from the API', () => {
    const fieldError = fieldErrorOf(new Error(faker.lorem.sentence()), 'name');

    expect(fieldError).toBeUndefined();
  });
});

describe('isGatewayError', () => {
  it.each([502, 503, 504])('reports the gateway error %i of a reverse proxy', (status) => {
    const gatewayError = isGatewayError(status);

    expect(gatewayError).toBe(true);
  });

  it('does not report another status', () => {
    const gatewayError = isGatewayError(500);

    expect(gatewayError).toBe(false);
  });
});
