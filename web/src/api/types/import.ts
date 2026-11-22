/** The character encoding of an imported or exported table. */
export const TableEncoding = {
  Utf8: 'utf-8',
  Windows1252: 'windows-1252',
} as const;
export type TableEncoding = (typeof TableEncoding)[keyof typeof TableEncoding];

/** Separates the values of an imported or exported table. */
export const TableSeparator = {
  Semicolon: 'semicolon',
  Comma: 'comma',
  Tab: 'tab',
} as const;
export type TableSeparator = (typeof TableSeparator)[keyof typeof TableSeparator];

/** The format of an imported or exported table. An empty field is detected on import. */
export type TableFormat = {
  encoding: TableEncoding | '';
  separator: TableSeparator | '';
};

/**
 * Assigns the columns of an imported table to the columns of its target, by the key of the target column,
 * and translates the values of the file into the values of a column, by its key and the value in the file.
 */
export type ImportMapping = {
  format: TableFormat;
  columns: Record<string, string>;
  values: Record<string, Record<string, string>>;
};

/** Describes a column an imported table is mapped onto. */
export type ImportColumnResponse = {
  key: string;
  required: boolean;
  // The values the column accepts, empty for a column that accepts any text
  choices: string[];
  // Further values the values of a file can be translated into, while the column accepts any text
  suggestions: string[];
};

/**
 * The start of an imported table with a guessed mapping and the distinct values of every header with few of them,
 * so they can be translated.
 */
export type ImportPreviewResponse = {
  format: TableFormat;
  header: string[];
  sample: string[][];
  mapping: ImportMapping;
  values: Record<string, string[]>;
  columns: ImportColumnResponse[];
};
