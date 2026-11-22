import { objectKeys } from '@/lib/types/guards';

/**
 * A set filter of a list as TableToolbar shows it while the filters are collapsed:
 * its name, its value as text, and how to remove it.
 */
export type TableFilterPill = {
  key: string;
  label: string;
  value: string;
  remove: () => void;
};

/**
 * Returns the pill of a filter, or none while the filter has no value.
 * A list builds its pills by joining these in the order of its filter fields.
 */
export function tableFilterPill(
  key: string,
  label: string,
  value: string | undefined,
  remove: () => void,
): TableFilterPill[] {
  if (value === undefined || value === '') return [];
  return [
    {
      key,
      label,
      value,
      remove,
    },
  ];
}

/** The select value of a filter without a value, since Reka UI rejects an empty string. */
export const TABLE_FILTER_ALL = 'all';

/** Returns the label of the option with a value, or undefined without a match. */
export function tableOptionLabel<V>(
  items: readonly { label: string; value: V }[],
  value: V | undefined,
): string | undefined {
  if (value === undefined) return undefined;
  return items.find((item) => item.value === value)?.label;
}

/** A date range filter with both ends as 'YYYY-MM-DD', where an empty end is open. */
export type TableDateRange = {
  from: string;
  to: string;
};

/** Returns a date range filter as text, with the year once for a range within one year. */
export function tableDateRangeLabel(
  range: TableDateRange,
  formatDate: (value: string, withYear: boolean) => string,
  labels: { from: string; to: string },
): string | undefined {
  const { from, to } = range;
  if (from && to) {
    return `${formatDate(from, from.slice(0, 4) !== to.slice(0, 4))} – ${formatDate(to, true)}`;
  }
  if (from) return `${labels.from} ${formatDate(from, true)}`;
  if (to) return `${labels.to} ${formatDate(to, true)}`;
  return undefined;
}

/** Leaves out the filters without a value, so an unset fixed filter keeps the one of the URL. */
export function withoutEmpty<T extends Record<string, unknown>>(values: T): Partial<T> {
  const result: Partial<T> = {};
  for (const key of objectKeys(values)) {
    const value = values[key];
    if (value !== undefined && value !== '') result[key] = value;
  }
  return result;
}
