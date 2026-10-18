export type ApiFieldError = {
  tag: string;
  param?: string;
};

export type FieldErrors = Record<string, ApiFieldError[]>;

export type ApiErrorResponse = {
  code: string;
  message: string;
  errors?: FieldErrors;
  requestId?: string;
};

export class ApiError extends Error {
  status: number;
  code: string;
  fields?: FieldErrors;
  // Seconds until a rate-limited request may be retried, from the Retry-After header.
  retryAfter?: number;
  // Matches the X-Request-ID header and the server's log lines of the request.
  requestId?: string;

  constructor(params: {
    message: string;
    status: number;
    code: string;
    fields?: FieldErrors;
    retryAfter?: number;
    requestId?: string;
  }) {
    super(params.message);
    this.status = params.status;
    this.code = params.code;
    this.fields = params.fields;
    this.retryAfter = params.retryAfter;
    this.requestId = params.requestId;
  }
}
