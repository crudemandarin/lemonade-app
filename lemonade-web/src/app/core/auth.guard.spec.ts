import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import {
  ActivatedRouteSnapshot,
  CanActivateFn,
  Router,
  RouterStateSnapshot,
  provideRouter,
} from '@angular/router';

import { authGuard, guestGuard, onboardingGuard, signedOutGuard } from './auth.guard';
import { SessionService, USERNAME_KEY } from './session.service';
import { FakeAuthPort, provideFakeAuth } from './testing/fake-auth';

describe('route guards', () => {
  let port: FakeAuthPort;
  let http: HttpTestingController;
  let router: Router;

  beforeEach(() => {
    port = new FakeAuthPort();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeAuth(port),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
  });

  afterEach(() => {
    http.verify();
    localStorage.removeItem(USERNAME_KEY);
  });

  const asGuest = () => TestBed.inject(SessionService).signIn('lemonjoe');

  /** Runs a guard and answers /api/me if it asks (profile: a username, 'none' or 'error'). */
  async function run(guard: CanActivateFn, profile?: string): Promise<boolean | string> {
    const result = TestBed.runInInjectionContext(() =>
      guard({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot),
    ) as Promise<boolean | ReturnType<Router['createUrlTree']>>;
    await new Promise((resolve) => setTimeout(resolve));
    if (profile) {
      const req = http.expectOne('/api/me');
      if (profile === 'none') {
        req.flush({ error: 'profile_required', message: 'x' }, { status: 403, statusText: '' });
      } else if (profile === 'error') {
        req.flush({}, { status: 500, statusText: '' });
      } else {
        req.flush({ id: 1, username: profile });
      }
    }
    const value = await result;
    return typeof value === 'boolean' ? value : router.serializeUrl(value);
  }

  describe('authGuard', () => {
    it('waits for the stored session before deciding', async () => {
      port.holdFirstEvent = true;
      const pending = TestBed.runInInjectionContext(() =>
        authGuard({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot),
      ) as Promise<unknown>;
      let settled = false;
      pending.then(() => (settled = true));
      await new Promise((resolve) => setTimeout(resolve));
      expect(settled).toBeFalse();

      port.emit(null);
      await pending;
      expect(settled).toBeTrue();
    });

    it('sends a signed-out visitor to /signin', async () => {
      expect(await run(authGuard)).toBe('/signin');
    });

    it('lets a signed-in player with a profile through', async () => {
      port.user = { uid: 'u1' };
      expect(await run(authGuard, 'lemonjoe')).toBeTrue();
    });

    it('lets a guest with a stored username through', async () => {
      asGuest();
      expect(await run(authGuard)).toBeTrue();
    });

    it('sends a signed-in account with no profile to choose a username', async () => {
      port.user = { uid: 'u1' };
      expect(await run(authGuard, 'none')).toBe('/signin/username');
    });

    it('sends the player to /signin when the profile cannot be loaded', async () => {
      port.user = { uid: 'u1' };
      expect(await run(authGuard, 'error')).toBe('/signin');
    });
  });

  describe('signedOutGuard', () => {
    it('shows the sign-in page to a signed-out visitor', async () => {
      expect(await run(signedOutGuard)).toBeTrue();
    });

    it('sends a signed-in player on to the game', async () => {
      port.user = { uid: 'u1' };
      expect(await run(signedOutGuard, 'lemonjoe')).toBe('/game');
    });

    it('sends a guest straight to the game', async () => {
      asGuest();
      expect(await run(signedOutGuard)).toBe('/game');
    });

    it('sends a signed-in account with no profile to onboarding', async () => {
      port.user = { uid: 'u1' };
      expect(await run(signedOutGuard, 'none')).toBe('/signin/username');
    });
  });

  describe('onboardingGuard', () => {
    it('needs a signed-in account', async () => {
      expect(await run(onboardingGuard)).toBe('/signin');
    });

    it('shows onboarding to an account with no profile', async () => {
      port.user = { uid: 'u1' };
      expect(await run(onboardingGuard, 'none')).toBeTrue();
    });

    it('skips onboarding for a player who already has a profile', async () => {
      port.user = { uid: 'u1' };
      expect(await run(onboardingGuard, 'lemonjoe')).toBe('/game');
    });
  });

  describe('guestGuard', () => {
    it('shows the secure page to a guest', async () => {
      asGuest();
      expect(await run(guestGuard)).toBeTrue();
    });

    it('sends a signed-out visitor to sign in', async () => {
      expect(await run(guestGuard)).toBe('/signin');
    });

    it('has nothing to offer an account that is already secured', async () => {
      port.user = { uid: 'u1' };
      TestBed.inject(SessionService).signInSecured('lemonjoe');
      expect(await run(guestGuard)).toBe('/game');
    });
  });
});
