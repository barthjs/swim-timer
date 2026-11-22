import type { ApiFieldError, FieldErrors } from '@/api/client/errors';
import type { ImportColumnResponse, ImportMapping, TableFormat } from '@/api/types/import';

/**
 * Stands for a column the mapping leaves out and a value it keeps,
 * since a select cannot hold an empty value.
 */
export const IMPORT_UNSET = '\u0000unset';

/** Returns a format whose encoding and separator are detected. */
export function emptyTableFormat(): TableFormat {
  return {
    encoding: '',
    separator: '',
  };
}

/** Returns the values the values of a file can be translated into for a column, none for free text. */
export function importTargets(column: ImportColumnResponse): string[] {
  return column.choices.length ? column.choices : column.suggestions;
}

/**
 * Returns the values of the file a column with targets can translate, in the order they occur,
 * or none for a column without targets, without an assigned header, or with a header of many values.
 */
export function importValueSources(
  column: ImportColumnResponse,
  mapping: ImportMapping,
  values: Record<string, string[]>,
): string[] {
  if (!importTargets(column).length) return [];
  const header = mapping.columns[column.key];
  if (!header) return [];
  return values[header] ?? [];
}

/**
 * Returns the mapping as the server expects it: without the columns it leaves out,
 * and with only the translations of assigned columns that change a value.
 */
export function cleanImportMapping(mapping: ImportMapping): ImportMapping {
  const columns: Record<string, string> = {};
  for (const [key, header] of Object.entries(mapping.columns)) {
    if (header && header !== IMPORT_UNSET) columns[key] = header;
  }

  const values: Record<string, Record<string, string>> = {};
  for (const [key, translations] of Object.entries(mapping.values)) {
    if (!columns[key]) continue;
    const kept = Object.entries(translations).filter(
      ([value, target]) => target !== IMPORT_UNSET && target !== value,
    );
    if (kept.length) values[key] = Object.fromEntries(kept);
  }

  return {
    format: {
      ...mapping.format,
    },
    columns,
    values,
  };
}

/** A failed row of an import with the failures of its columns, under '' for the whole row. */
export type ImportRowError = {
  line: number;
  fields: Record<string, ApiFieldError[]>;
};

/** The validation errors of an import, split by what they belong to. */
export type ImportErrors = {
  // The errors of the other fields of the form, such as a name
  form: FieldErrors;
  file: ApiFieldError[];
  columns: Record<string, ApiFieldError[]>;
  values: Record<string, ApiFieldError[]>;
  rows: ImportRowError[];
};

/** Matches the field of a failed row, rows.<line> with an optional .<key>. */
const ROW_FIELD_PATTERN = /^rows\.(\d+)(?:\.(.+))?$/;

/**
 * Sorts the field errors of a failed import into the errors of the file,
 * the mapping of the columns and values, the rows by line, and the remaining fields of the form.
 */
export function splitImportErrors(fields: FieldErrors): ImportErrors {
  const result: ImportErrors = {
    form: {},
    file: [],
    columns: {},
    values: {},
    rows: [],
  };

  for (const [field, errors] of Object.entries(fields)) {
    const row = ROW_FIELD_PATTERN.exec(field);
    if (row?.[1]) {
      const line = Number(row[1]);
      let entry = result.rows.find((item) => item.line === line);
      if (!entry) {
        entry = {
          line,
          fields: {},
        };
        result.rows.push(entry);
      }
      entry.fields[row[2] ?? ''] = errors;
      continue;
    }
    if (field === 'file') {
      result.file = errors;
    } else if (field.startsWith('mapping.columns.')) {
      result.columns[field.slice('mapping.columns.'.length)] = errors;
    } else if (field.startsWith('mapping.values.')) {
      result.values[field.slice('mapping.values.'.length)] = errors;
    } else {
      result.form[field] = errors;
    }
  }

  result.rows.sort((a, b) => a.line - b.line);
  return result;
}
