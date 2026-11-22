type Primitive = string | number | boolean | null | undefined;

/** Every dot-separated path into the fields of T, including nested objects. */
export type Paths<T> = T extends Primitive
  ? never
  : {
      [K in keyof T & string]: T[K] extends Primitive ? K : K | `${K}.${Paths<T[K]>}`;
    }[keyof T & string];

/** Reads a possibly nested field of source by a dot-separated path. */
export function getAtPath(source: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((value, segment) => {
    if (value !== null && typeof value === 'object' && segment in value) {
      const field: unknown = Reflect.get(value, segment);
      return field;
    }
    return undefined;
  }, source);
}

/** Writes a possibly nested field of target by a dot-separated path whose parent objects exist. */
export function setAtPath(target: object, path: string, value: unknown): void {
  const segments = path.split('.');
  const field = segments.pop();
  const parent = segments.length > 0 ? getAtPath(target, segments.join('.')) : target;
  if (field === undefined || parent === null || typeof parent !== 'object') return;
  Reflect.set(parent, field, value);
}

/** Returns the dot-separated paths of the leaf values that differ between before and after. */
export function changedPaths(before: unknown, after: unknown, prefix = ''): string[] {
  if (!isBranch(before) && !isBranch(after)) {
    return Object.is(before, after) ? [] : [prefix];
  }

  const keys = new Set([...branchKeys(before), ...branchKeys(after)]);
  return [...keys].flatMap((key) =>
    changedPaths(getAtPath(before, key), getAtPath(after, key), prefix ? `${prefix}.${key}` : key),
  );
}

/** Reports whether a value is an object or array whose fields have paths of their own. */
function isBranch(value: unknown): value is object {
  return value !== null && typeof value === 'object';
}

/** Returns the keys of a branch, or none for a leaf value. */
function branchKeys(value: unknown): string[] {
  return isBranch(value) ? Object.keys(value) : [];
}
