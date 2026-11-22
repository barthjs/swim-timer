import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client/client';
import type { TableFormat, ImportPreviewResponse } from '@/api/types/import';
import type {
  RecordTableCreateRequest,
  RecordTableImportRequest,
  RecordTableListResponse,
  RecordTableResponse,
  RecordTableUpdateRequest,
} from '@/api/types/recordTable';

/** Selects the format and the columns of an export, every column without any. */
export type RecordTableExport = {
  format: TableFormat;
  columns: string[];
};

/** Returns the multipart body of an upload of a table with the request as JSON in its data field. */
function importBody(file: File, request: unknown): FormData {
  const body = new FormData();
  body.append('file', file);
  body.append('data', JSON.stringify(request));
  return body;
}

export const recordTableRepository = {
  /** Returns every record table in the order they are checked. */
  async list(): Promise<RecordTableListResponse> {
    return apiFetch('/record-tables', {
      method: 'GET',
    });
  },

  /** Stores a new record table with the records of a file. */
  async create(file: File, payload: RecordTableCreateRequest): Promise<RecordTableResponse> {
    return apiFetch('/record-tables', {
      method: 'POST',
      body: importBody(file, payload),
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** Replaces the name, the abbreviation, and the state of the record table with the given id. */
  async update(id: string, payload: RecordTableUpdateRequest): Promise<RecordTableResponse> {
    return apiFetch(`/record-tables/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** Removes the record table with the given id. */
  async delete(id: string): Promise<void> {
    return apiFetch(`/record-tables/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },

  /** Reads the start of a file and guesses its mapping, until signal cancels the reading. */
  async preview(
    file: File,
    format: TableFormat,
    signal?: AbortSignal,
  ): Promise<ImportPreviewResponse> {
    return apiFetch('/record-tables/preview', {
      method: 'POST',
      body: importBody(file, {
        format,
      }),
      timeout: UPLOAD_TIMEOUT_MS,
      signal,
    });
  },

  /** Replaces the records of the record table with the given id with the records of a file. */
  async import(
    id: string,
    file: File,
    payload: RecordTableImportRequest,
  ): Promise<RecordTableResponse> {
    return apiFetch(`/record-tables/${encodeURIComponent(id)}/import`, {
      method: 'POST',
      body: importBody(file, payload),
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** Puts the record tables in the order of the given ids. */
  async reorder(ids: string[]): Promise<void> {
    return apiFetch('/record-tables/order', {
      method: 'PUT',
      body: {
        ids,
      },
    });
  },

  /** Reads every record table again. */
  async reload(): Promise<void> {
    return apiFetch('/record-tables/reload', {
      method: 'POST',
    });
  },

  /** Returns the records of a table as a file in the given format and columns. */
  async export(id: string, options: RecordTableExport): Promise<Blob> {
    const query = new URLSearchParams();
    if (options.format.encoding) query.set('encoding', options.format.encoding);
    if (options.format.separator) query.set('separator', options.format.separator);
    if (options.columns.length) query.set('columns', options.columns.join(','));
    const search = query.toString();
    return apiFetch(
      `/record-tables/${encodeURIComponent(id)}/export${search ? `?${search}` : ''}`,
      {
        method: 'GET',
        responseType: 'blob',
      },
    );
  },
};
