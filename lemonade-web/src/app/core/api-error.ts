import { HttpErrorResponse } from '@angular/common/http';

/** Turns an HttpClient error into a message, preferring the API's `{ error, message }` body. */
export function apiErrorMessage(err: unknown): string {
  if (!(err instanceof HttpErrorResponse)) {
    return 'Unexpected error.';
  }
  if (err.error && typeof err.error === 'object') {
    if ('message' in err.error && err.error.message) {
      return String(err.error.message);
    }
    if ('error' in err.error && err.error.error) {
      return String(err.error.error);
    }
  }
  if (err.status === 0) {
    return 'Could not reach the server.';
  }
  return `Request failed with status ${err.status}.`;
}
