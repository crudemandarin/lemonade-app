import { HttpErrorResponse, HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, from, switchMap, throwError } from 'rxjs';

import { AuthService } from './auth.service';
import { SessionService } from './session.service';

const withToken = (req: HttpRequest<unknown>, token: string | null) =>
  token ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }) : req;

/**
 * Identifies the player to the API. Signed in with Google: the Firebase ID token as a
 * bearer header; a 401 may only mean it just expired, so refresh it and retry once (a
 * second 401 goes on to unauthorizedInterceptor, which signs out). Otherwise a guest
 * sends X-Username, which the API accepts only for accounts nobody has secured.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith('/api')) {
    return next(req);
  }
  const auth = inject(AuthService);
  if (!auth.firebaseUser()) {
    const username = inject(SessionService).username();
    return next(username ? req.clone({ setHeaders: { 'X-Username': username } }) : req);
  }
  return from(auth.getIdToken()).pipe(
    switchMap((token) =>
      next(withToken(req, token)).pipe(
        catchError((err: unknown) => {
          if (!(err instanceof HttpErrorResponse) || err.status !== 401 || !token) {
            return throwError(() => err);
          }
          return from(auth.getIdToken(true)).pipe(
            switchMap((fresh) => next(withToken(req, fresh))),
          );
        }),
      ),
    ),
  );
};
