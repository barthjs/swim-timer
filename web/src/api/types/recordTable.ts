import type { ImportMapping } from '@/api/types/import';

/** A new record table with the mapping of the file its records are imported from. */
export type RecordTableCreateRequest = {
  name: string;
  abbreviation: string;
  enabled: boolean;
  mapping: ImportMapping;
};

/** Replaces the name, the abbreviation, and the state of a record table. */
export type RecordTableUpdateRequest = {
  name: string;
  abbreviation: string;
  enabled: boolean;
};

/** Holds the mapping of a file whose records replace the records of a table. */
export type RecordTableImportRequest = {
  mapping: ImportMapping;
};

export type RecordTableResponse = {
  id: string;
  name: string;
  abbreviation: string;
  enabled: boolean;
  recordCount: number;
};

/** Every record table in the order they are checked. */
export type RecordTableListResponse = {
  loadedAt?: string;
  tables: RecordTableResponse[];
};
