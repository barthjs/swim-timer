/**
 * TableFilterPill is a set filter of a list as TableToolbar shows it while the filters are collapsed:
 * its name, its value as text, and how to remove it.
 */
export type TableFilterPill = {
  key: string;
  label: string;
  value: string;
  remove: () => void;
};

/**
 * tableFilterPill returns the pill of a filter, or none while the filter has no value.
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

/** tableOptionLabel returns the label of the option with a value, or undefined without a match. */
export function tableOptionLabel<V>(
  items: readonly { label: string; value: V }[],
  value: V | undefined,
): string | undefined {
  if (value === undefined) return undefined;
  return items.find((item) => item.value === value)?.label;
}

/** TableDateRange is a date range filter with both ends as 'YYYY-MM-DD', where an empty end is open. */
export type TableDateRange = {
  from: string;
  to: string;
};

/**
 * tableDateRangeLabel returns a date range filter as text, such as "01.05.2026 – 31.05.2026",
 * or undefined while neither end is set. An open range names its one end with the given labels.
 */
export function tableDateRangeLabel(
  range: TableDateRange,
  formatDate: (value: string) => string,
  labels: { from: string; to: string },
): string | undefined {
  const { from, to } = range;
  if (from && to) return `${formatDate(from)} – ${formatDate(to)}`;
  if (from) return `${labels.from} ${formatDate(from)}`;
  if (to) return `${labels.to} ${formatDate(to)}`;
  return undefined;
}
