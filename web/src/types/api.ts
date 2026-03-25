export interface ValidationErrorDetail {
  rule: string;
  params?: Record<string, string>;
}

export interface ValidationErrorResponse {
  message: string;
  errors: Record<string, ValidationErrorDetail>;
}

export interface ApiErrorResponse {
  message: string;
  errors?: Record<string, ValidationErrorDetail>;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export class ValidationError extends ApiError {
  constructor(
    public errors: Record<string, ValidationErrorDetail>,
    message: string = 'Validation failed',
  ) {
    super(422, message);
    this.name = 'ValidationError';
  }
}

export class NotFoundError extends ApiError {
  constructor(message: string = 'Resource not found') {
    super(404, message);
    this.name = 'NotFoundError';
  }
}

export class UnauthorizedError extends ApiError {
  constructor(message: string = 'Unauthorized') {
    super(401, message);
    this.name = 'UnauthorizedError';
  }
}

export class AuthenticationError extends ApiError {
  constructor(message: string = 'Authentication failed') {
    super(401, message);
    this.name = 'AuthenticationError';
  }
}

export class BadRequestError extends ApiError {
  constructor(message: string = 'Bad request') {
    super(400, message);
    this.name = 'BadRequestError';
  }
}

export class ServerError extends ApiError {
  constructor(message: string = 'Internal server error') {
    super(500, message);
    this.name = 'ServerError';
  }
}

export function isValidationError(error: unknown): error is ValidationError {
  return error instanceof ValidationError;
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export function getErrorMessageKey(error: unknown): string {
  if (error instanceof AuthenticationError) {
    return 'errors.authenticationFailed';
  }
  if (error instanceof UnauthorizedError) {
    return 'errors.unauthorized';
  }
  if (error instanceof NotFoundError) {
    return 'errors.notFound';
  }
  if (error instanceof BadRequestError) {
    return 'errors.badRequest';
  }
  if (error instanceof ServerError) {
    return 'errors.serverError';
  }
  if (error instanceof ApiError) {
    return 'errors.unknown';
  }
  return 'errors.unknown';
}
