import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';

import { SessionService } from './session.service';

/** Identifies the player to the API. Intentionally not secure (username-only login). */
export const usernameInterceptor: HttpInterceptorFn = (req, next) => {
  const username = inject(SessionService).username();
  return next(username ? req.clone({ setHeaders: { 'X-Username': username } }) : req);
};
