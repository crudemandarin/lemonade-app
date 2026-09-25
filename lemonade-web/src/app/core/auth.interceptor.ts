import { HttpErrorResponse, HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, from, switchMap, throwError } from 'rxjs';

import { AuthService } from './auth.service';

const withToken = (req: HttpRequest<unknown>, token: string | null) =>
  token ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }) : req;

/**
 * Sends the Firebase ID token on every API call. A 401 may only mean the token just
 * expired, so refresh it and retry once; if that fails too the error goes on to
 * unauthorizedInterceptor, which signs the player out.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  if (!req.url.startsWith('/api')) {
    return next(req);
  }
  const auth = inject(AuthService);
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
