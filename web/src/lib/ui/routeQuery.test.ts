import { faker } from '@faker-js/faker';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { queryParam, replaceRouteQuery } from '@/lib/ui/routeQuery';
import { fakeRouter } from '@/test/router';

describe('queryParam', () => {
  it('returns the value of a parameter', () => {
    const value = faker.lorem.word();

    expect(queryParam(value)).toBe(value);
  });

  it('returns the first value of a repeated parameter', () => {
    const [first, second] = [faker.lorem.word(), faker.lorem.word()];

    expect(queryParam([first, second])).toBe(first);
  });

  it('returns undefined for a missing, empty, or valueless parameter', () => {
    expect([queryParam(undefined), queryParam(''), queryParam(null), queryParam([])]).toEqual([
      undefined,
      undefined,
      undefined,
      undefined,
    ]);
  });
});

describe('replaceRouteQuery', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('replaces the query with the result of the change', async () => {
    const fake = fakeRouter();
    const value = faker.lorem.word();

    await replaceRouteQuery(fake.route, fake.router, () => ({
      key: value,
    }));

    expect(fake.route.query).toEqual({
      key: value,
    });
  });

  it('keeps the hash of the page', async () => {
    const fake = fakeRouter();
    const hash = `#${faker.lorem.word()}`;
    vi.stubGlobal('window', {
      location: {
        hash,
      },
    });

    await replaceRouteQuery(fake.route, fake.router, () => ({}));

    expect(fake.router.replace).toHaveBeenCalledWith({
      query: {},
      hash,
    });
  });

  it('runs a change on the query the change before it left', async () => {
    const fake = fakeRouter();
    const [first, second] = [faker.lorem.word(), faker.lorem.word()];
    void replaceRouteQuery(fake.route, fake.router, (query) => ({
      ...query,
      first,
    }));

    await replaceRouteQuery(fake.route, fake.router, (query) => ({
      ...query,
      second,
    }));

    expect(fake.route.query).toEqual({
      first,
      second,
    });
  });

  it('drops a change when the page changed before it runs', async () => {
    const fake = fakeRouter();
    const pending = replaceRouteQuery(fake.route, fake.router, () => ({
      key: faker.lorem.word(),
    }));
    fake.route.path = `/${faker.lorem.word()}`;

    await pending;

    expect(fake.router.replace).not.toHaveBeenCalled();
    expect(fake.route.query).toEqual({});
  });

  it('runs a change after one that failed', async () => {
    const fake = fakeRouter();
    const value = faker.lorem.word();
    fake.router.replace.mockRejectedValueOnce(new Error(faker.lorem.sentence()));
    void replaceRouteQuery(fake.route, fake.router, () => ({})).catch(() => undefined);

    await replaceRouteQuery(fake.route, fake.router, () => ({
      key: value,
    }));

    expect(fake.route.query).toEqual({
      key: value,
    });
  });
});
