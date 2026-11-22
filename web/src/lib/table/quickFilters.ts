/** The values of the filters of a list, a filter without a value is unset. */
export type TableFilterValues = Record<string, string | number | boolean | undefined>;

/**
 * A named set of filter values that a list offers above its rows.
 * Every filter it leaves out is unset while it applies.
 */
export type TableQuickFilter<TFilters extends TableFilterValues = TableFilterValues> = {
  id: string;
  label: string;
  icon?: string;
  filters: Partial<TFilters>;
};

/** Reports whether two sets of filters set the same filters to the same values. */
export function sameTableFilters(a: TableFilterValues, b: TableFilterValues): boolean {
  const left = setFilterValues(a);
  const right = setFilterValues(b);
  const keys = Object.keys(left);
  return keys.length === Object.keys(right).length && keys.every((key) => left[key] === right[key]);
}

/** Returns the filters that have a value, each as the text the URL holds. */
function setFilterValues(values: TableFilterValues): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(values)) {
    if (value === undefined || value === '') continue;
    result[key] = String(value);
  }
  return result;
}
