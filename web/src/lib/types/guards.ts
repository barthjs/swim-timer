/** Reports whether a value is a plain object whose fields can be read by name. */
export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** Reports whether a value is one of the allowed values. */
export function isOneOf<T>(allowed: readonly T[], value: unknown): value is T {
  return allowed.some((candidate) => candidate === value);
}

/** Returns the own keys of an object, typed as the keys of its type. */
export function objectKeys<T extends object>(value: T): (keyof T & string)[] {
  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- Object.keys returns the own string keys of the value
  return Object.keys(value) as (keyof T & string)[];
}
