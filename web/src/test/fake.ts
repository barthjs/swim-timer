import { faker } from '@faker-js/faker';

/** A tuple of N values of T. */
type FakeTuple<T, N extends number, R extends T[] = []> = R['length'] extends N
  ? R
  : FakeTuple<T, N, [...R, T]>;

/** Returns count different values of a faker generator, for values a test must not mix up. */
export function fakeDistinct<T, N extends number>(make: () => T, count: N): FakeTuple<T, N> {
  return faker.helpers.uniqueArray(make, count) as FakeTuple<T, N>;
}
