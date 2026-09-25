import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { tap } from 'rxjs';

import { AuthService, apiErrorCode } from './auth.service';

/**
 * Runs outside authInterceptor, so a 401 here already survived one token refresh: the
 * sign-in is no good. Sign out (Firebase too) and go to the sign-in page. A 403
 * `profile_required` is not a sign-out: the Google account is fine, it just has no
 * username yet, so send the player to choose or claim one. (Except on `GET /api/me`,
 * which the guards call to find that out; redirecting there would loop.)
 */
export const unauthorizedInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  return next(req).pipe(
    tap({
      error: (err: unknown) => {
        if (!(err instanceof HttpErrorResponse)) {
          return;
        }
        if (err.status === 401) {
          void auth.signOut();
          router.navigateByUrl('/signin');
        } else if (
          err.status === 403 &&
          apiErrorCode(err) === 'profile_required' &&
          req.url !== '/api/me' // the profile probe: its 403 is an answer the guards act on
        ) {
          router.navigateByUrl('/signin/username');
        }
      },
    }),
  );
};
