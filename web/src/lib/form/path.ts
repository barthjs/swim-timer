type Primitive = string | number | boolean | null | undefined;

/** Paths are every dot-separated path into the fields of T, including nested objects. */
export type Paths<T> = T extends Primitive
  ? never
  : {
      [K in keyof T & string]: T[K] extends Primitive ? K : K | `${K}.${Paths<T[K]>}`;
    }[keyof T & string];

/** getAtPath reads a possibly nested field of source by a dot-separated path. */
export function getAtPath(source: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((value, segment) => {
    if (value !== null && typeof value === 'object' && segment in value) {
      return (value as Record<string, unknown>)[segment];
    }
    return undefined;
  }, source);
}

/** setAtPath writes a possibly nested field of target by a dot-separated path whose parent objects exist. */
export function setAtPath(target: object, path: string, value: unknown): void {
  const segments = path.split('.');
  const field = segments.pop();
  const parent = segments.length > 0 ? getAtPath(target, segments.join('.')) : target;
  if (field === undefined || parent === null || typeof parent !== 'object') return;
  (parent as Record<string, unknown>)[field] = value;
}
