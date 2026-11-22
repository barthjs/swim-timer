import { describe, expect, it } from 'vitest';
import { faker } from '@faker-js/faker';

import { isOneOf, isRecord, objectKeys } from '@/lib/types/guards';

describe('isRecord', () => {
  it('accepts a plain object', () => {
    expect(
      isRecord({
        name: faker.lorem.word(),
      }),
    ).toBe(true);
  });

  it.each([
    {
      name: 'null',
      value: null,
    },
    {
      name: 'an array',
      value: [faker.lorem.word()],
    },
    {
      name: 'a text',
      value: faker.lorem.word(),
    },
    {
      name: 'undefined',
      value: undefined,
    },
  ])('rejects $name', ({ value }) => {
    expect(isRecord(value)).toBe(false);
  });
});

describe('isOneOf', () => {
  const allowed = ['first', 'second'] as const;

  it('accepts an allowed value', () => {
    expect(isOneOf(allowed, 'second')).toBe(true);
  });

  it('rejects any other value', () => {
    expect(isOneOf(allowed, faker.string.uuid())).toBe(false);
  });

  it('rejects a value of another type', () => {
    expect(isOneOf(allowed, faker.number.int())).toBe(false);
  });
});

describe('objectKeys', () => {
  it('returns the own keys of an object', () => {
    const value = {
      name: faker.lorem.word(),
      count: faker.number.int(),
    };

    expect(objectKeys(value)).toEqual(['name', 'count']);
  });
});
