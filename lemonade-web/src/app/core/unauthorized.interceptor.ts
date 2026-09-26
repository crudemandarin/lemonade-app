import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { tap } from 'rxjs';

import { SessionService } from './session.service';

/**
 * A 401 means the stored username is unknown to the server (for example after a
 * database reset). Drop it and send the player to sign in again.
 */
export const unauthorizedInterceptor: HttpInterceptorFn = (req, next) => {
  const session = inject(SessionService);
  const router = inject(Router);
  return next(req).pipe(
    tap({
      error: (err: unknown) => {
        if (err instanceof HttpErrorResponse && err.status === 401) {
          session.signOut();
          router.navigateByUrl('/signin');
        }
      },
    }),
  );
};
