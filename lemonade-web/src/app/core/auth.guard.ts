import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthService } from './auth.service';

/**
 * Game pages need a signed-in player with a profile. Waits for the stored session to
 * be checked, so a reload never flashes the sign-in page.
 */
export const authGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  await auth.whenReady();
  if (!auth.firebaseUser()) {
    return router.createUrlTree(['/signin']);
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
  await auth.whenReady();
  if (!auth.firebaseUser()) {
    return true;
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
