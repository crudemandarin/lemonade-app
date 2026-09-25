import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { tap } from 'rxjs';

import { AuthService, apiErrorCode } from './auth.service';

/**
 * Runs outside authInterceptor, so a 401 here already survived one token refresh: the
 * sign-in is no good. Sign out and go to the sign-in page. Two cases are not a plain
 * sign-out:
 *  - 401 `account_secured`: this guest name was secured with Google elsewhere. Forget the
 *    stored name (leave any Google session alone) and say why on the sign-in page.
 *  - 403 `profile_required`: the Google account is fine, it just has no username yet, so
 *    send the player to choose or claim one. (Except on `GET /api/me`, which the guards
 *    call to find that out; redirecting there would loop.)
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
        const code = apiErrorCode(err);
        if (err.status === 401 && code === 'account_secured') {
          auth.forgetGuest();
          router.navigateByUrl('/signin?reason=secured');
        } else if (err.status === 401) {
          void auth.signOut();
          router.navigateByUrl('/signin');
        } else if (err.status === 403 && code === 'profile_required' && req.url !== '/api/me') {
          router.navigateByUrl('/signin/username');
        }
      },
    }),
  );
};
