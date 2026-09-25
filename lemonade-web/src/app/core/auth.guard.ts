import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthService } from './auth.service';
import { SessionService } from './session.service';

/**
 * Game pages need a player: a guest with a stored username, or a Google sign-in with a
 * profile. Waits for the stored session to be checked, so a reload never flashes the
 * sign-in page.
 */
export const authGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const session = inject(SessionService);
  await auth.whenReady();
  if (!auth.firebaseUser()) {
    return session.username() ? true : router.createUrlTree(['/signin']);
  }
  try {
    return (await auth.loadProfile()) ? true : router.createUrlTree(['/signin', 'username']);
  } catch {
    return router.createUrlTree(['/signin']);
  }
};

/** The sign-in page is for signed-out players; anyone else is sent on to where they belong. */
export const signedOutGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const session = inject(SessionService);
  await auth.whenReady();
  if (!auth.firebaseUser()) {
    return session.username() ? router.createUrlTree(['/game']) : true;
  }
  try {
    return router.createUrlTree((await auth.loadProfile()) ? ['/game'] : ['/signin', 'username']);
  } catch {
    return true;
  }
};

/** Onboarding is for signed-in players who have no username yet. */
export const onboardingGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  await auth.whenReady();
  if (!auth.firebaseUser()) {
    return router.createUrlTree(['/signin']);
  }
  try {
    return (await auth.loadProfile()) ? router.createUrlTree(['/game']) : true;
  } catch {
    return true;
  }
};

/** Securing an account is for guests; a secured player has nothing to do there. */
export const guestGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const session = inject(SessionService);
  await auth.whenReady();
  if (session.secured()) {
    return router.createUrlTree(['/game']);
  }
  return session.username() ? true : router.createUrlTree(['/signin']);
};
