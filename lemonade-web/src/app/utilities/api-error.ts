import { HttpErrorResponse } from '@angular/common/http';

/** Turns an HttpClient error into a message, preferring the API's `{ "error": ... }` body. */
export function apiErrorMessage(err: unknown): string {
  if (!(err instanceof HttpErrorResponse)) {
    return 'Unexpected error.';
  }
  if (err.error && typeof err.error === 'object' && 'error' in err.error) {
    return String(err.error.error);
  }
  if (err.status === 0) {
    return 'Could not reach the API.';
  }
  return `Request failed with status ${err.status}.`;
}
