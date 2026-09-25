import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { SessionService } from './session.service';

/** Sends players without a stored username to /signin. */
export const authGuard: CanActivateFn = () =>
  inject(SessionService).username() ? true : inject(Router).createUrlTree(['/signin']);
