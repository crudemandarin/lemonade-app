import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed, fakeAsync, tick } from '@angular/core/testing';

import {
  AuthService,
  IS_STANDALONE,
  LINKING_KEY,
  READY_TIMEOUT_MS,
  apiErrorCode,
} from './auth.service';
import { FirebaseConfigService } from './firebase-config';
import { SessionService, USERNAME_KEY } from './session.service';
import { FakeAuthPort, provideFakeAuth } from './testing/fake-auth';

const profileRequired = {
  error: 'profile_required',
  message: 'Choose a username to start playing.',
};

describe('AuthService', () => {
  let port: FakeAuthPort;
  let http: HttpTestingController;

  function setup(configure: (p: FakeAuthPort) => void = () => undefined, standalone = false) {
    port = new FakeAuthPort();
    configure(port);
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeAuth(port),
        { provide: IS_STANDALONE, useValue: () => standalone },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    return TestBed.inject(AuthService);
  }

  afterEach(() => {
    http?.verify();
    localStorage.removeItem(USERNAME_KEY);
    sessionStorage.removeItem(LINKING_KEY);
  });

  it('is not ready until the stored session has been checked', async () => {
    const auth = setup((p) => (p.holdFirstEvent = true));
    expect(auth.ready()).toBeFalse();

    port.emit(null);
    await auth.whenReady();

    expect(auth.ready()).toBeTrue();
    expect(auth.firebaseUser()).toBeNull();
  });

  it('stops waiting for a Firebase that never answers and treats the player as signed out', fakeAsync(() => {
    const auth = setup((p) => (p.holdFirstEvent = true));
    tick(READY_TIMEOUT_MS - 1);
    expect(auth.ready()).toBeFalse();

    tick(1);
    expect(auth.ready()).toBeTrue();
    expect(auth.firebaseUser()).toBeNull();

    // If it answers late, that still counts.
    port.emit({ uid: 'u1' });
    expect(auth.firebaseUser()).toEqual({ uid: 'u1' });
    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
  }));

  it('loads the profile of a restored session', async () => {
    const auth = setup((p) => (p.user = { uid: 'u1' }));
    await auth.whenReady();

    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
    await auth.loadProfile();

    expect(auth.profile()).toEqual({ username: 'lemonjoe' });
    expect(TestBed.inject(SessionService).username()).toBe('lemonjoe');
  });

  it('shares one profile request between callers', async () => {
    const auth = setup((p) => (p.user = { uid: 'u1' }));
    const a = auth.loadProfile();
    const b = auth.loadProfile();

    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });

    expect(await a).toBeTrue();
    expect(await b).toBeTrue();
  });

  it('reports a missing profile (403 profile_required) without signing out', async () => {
    const auth = setup((p) => (p.user = { uid: 'u1' }));
    const result = auth.loadProfile();
    http.expectOne('/api/me').flush(profileRequired, { status: 403, statusText: '' });

    expect(await result).toBeFalse();
    expect(auth.firebaseUser()).not.toBeNull();
    expect(auth.profile()).toBeNull();
    expect(port.signOutCalls).toBe(0);
  });

  it('rethrows other profile errors and tries again next time', async () => {
    const auth = setup((p) => (p.user = { uid: 'u1' }));
    const first = auth.loadProfile();
    http.expectOne('/api/me').flush({}, { status: 500, statusText: '' });
    await expectAsync(first).toBeRejected();

    const second = auth.loadProfile();
    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
    expect(await second).toBeTrue();
  });

  describe('signInWithGoogle', () => {
    it('uses the popup', async () => {
      const auth = setup();
      await auth.signInWithGoogle();
      expect(port.popupCalls).toBe(1);
      expect(port.redirectCalls).toBe(0);
      expect(auth.firebaseUser()).toEqual({ uid: 'uid-1' });
      http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
    });

    it('falls back to a redirect when the popup is blocked', async () => {
      const auth = setup((p) => (p.popupError = { code: 'auth/popup-blocked' }));
      await auth.signInWithGoogle();
      expect(port.redirectCalls).toBe(1);
    });

    it('goes straight to a redirect in an installed app', async () => {
      const auth = setup(() => undefined, true);
      await auth.signInWithGoogle();
      expect(port.popupCalls).toBe(0);
      expect(port.redirectCalls).toBe(1);
    });

    it('treats closing the popup as changing your mind', async () => {
      for (const code of ['auth/popup-closed-by-user', 'auth/cancelled-popup-request']) {
        const auth = setup((p) => (p.popupError = { code }));
        await expectAsync(auth.signInWithGoogle()).toBeResolved();
        expect(port.redirectCalls).toBe(0);
        TestBed.resetTestingModule();
      }
    });

    it('rethrows other failures', async () => {
      const auth = setup((p) => (p.popupError = { code: 'auth/network-request-failed' }));
      await expectAsync(auth.signInWithGoogle()).toBeRejected();
    });
  });

  it('signOut signs out of Firebase and forgets the profile', async () => {
    const auth = setup((p) => (p.user = { uid: 'u1' }));
    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
    await auth.loadProfile();
    expect(auth.profile()).not.toBeNull();

    await auth.signOut();

    expect(port.signOutCalls).toBe(1);
    expect(auth.firebaseUser()).toBeNull();
    expect(auth.profile()).toBeNull();
  });

  it('createProfile and claim set the profile from the response', async () => {
    const auth = setup();
    const created = auth.createProfile('lemonjoe');
    const req = http.expectOne('/api/me/username');
    expect(req.request.body).toEqual({ username: 'lemonjoe' });
    req.flush({ id: 1, username: 'lemonjoe' });
    await created;
    expect(auth.profile()).toEqual({ username: 'lemonjoe' });

    await auth.signOut();
    const claimed = auth.claim('oldtimer');
    const claimReq = http.expectOne('/api/me/claim');
    expect(claimReq.request.body).toEqual({ username: 'oldtimer' });
    claimReq.flush({ id: 2, username: 'oldtimer' });
    await claimed;
    expect(auth.profile()).toEqual({ username: 'oldtimer' });
  });

  it('asks the SDK for tokens and never keeps one', async () => {
    const auth = setup();
    expect(await auth.getIdToken()).toBe('token-1');
    expect(await auth.getIdToken(true)).toBe('token-2');
    expect(port.tokenRequests).toEqual([false, true]);
    expect(JSON.stringify(localStorage)).not.toContain('token-');
  });

  describe('guests', () => {
    it('keeps a stored guest when Firebase reports nobody signed in', async () => {
      localStorage.setItem(USERNAME_KEY, 'lemonjoe');
      const auth = setup();
      await auth.whenReady();

      expect(auth.firebaseUser()).toBeNull();
      expect(auth.profile()).toEqual({ username: 'lemonjoe' });
      expect(auth.secured()).toBeFalse();
    });

    it('signs a guest in with a username alone', async () => {
      const auth = setup();
      const done = auth.guestSignIn('lemonjoe');
      const req = http.expectOne('/api/login');
      expect(req.request.body).toEqual({ username: 'lemonjoe' });
      req.flush({ id: 1, username: 'lemonjoe' });
      await done;

      expect(auth.profile()).toEqual({ username: 'lemonjoe' });
      expect(localStorage.getItem(USERNAME_KEY)).toBe('lemonjoe');
    });

    it('rejects a protected username with account_secured and signs nobody in', async () => {
      const auth = setup();
      const done = auth.guestSignIn('lemonjoe');
      http
        .expectOne('/api/login')
        .flush({ error: 'account_secured', message: 'x' }, { status: 409, statusText: '' });

      await expectAsync(done).toBeRejected();
      expect(auth.profile()).toBeNull();
    });

    it('signOut forgets the guest too', async () => {
      localStorage.setItem(USERNAME_KEY, 'lemonjoe');
      const auth = setup();
      await auth.signOut();
      expect(auth.profile()).toBeNull();
      expect(localStorage.getItem(USERNAME_KEY)).toBeNull();
    });

    it('googleAvailable follows whether Firebase is configured', () => {
      const auth = setup();
      expect(auth.googleAvailable).toBeFalse();
      TestBed.inject(FirebaseConfigService).config = {
        apiKey: 'k',
        authDomain: 'a',
        projectId: 'p',
        appId: 'i',
      };
      expect(auth.googleAvailable).toBeTrue();
    });
  });

  describe('linkGoogle (securing a guest account)', () => {
    beforeEach(() => localStorage.setItem(USERNAME_KEY, 'lemonjoe'));

    it('signs in with Google and links the guest username', async () => {
      const auth = setup();
      const done = auth.linkGoogle();
      await new Promise((resolve) => setTimeout(resolve));
      const req = http.expectOne('/api/me/claim');
      expect(req.request.body).toEqual({ username: 'lemonjoe' });
      req.flush({ id: 1, username: 'lemonjoe' });
      await done;

      expect(auth.secured()).toBeTrue();
      expect(auth.profile()).toEqual({ username: 'lemonjoe' });
      expect(localStorage.getItem(USERNAME_KEY)).toBeNull();
      expect(sessionStorage.getItem(LINKING_KEY)).toBeNull();
    });

    it('stays a guest, signed out of Google, when the link fails', async () => {
      const auth = setup();
      const done = auth.linkGoogle();
      await new Promise((resolve) => setTimeout(resolve));
      http
        .expectOne('/api/me/claim')
        .flush({ error: 'already_linked', message: 'x' }, { status: 409, statusText: '' });

      await expectAsync(done).toBeRejected();
      expect(auth.secured()).toBeFalse();
      expect(auth.linkError()).toBe('already_linked');
      expect(port.signOutCalls).toBe(1);
      expect(localStorage.getItem(USERNAME_KEY)).toBe('lemonjoe');
    });

    it('does nothing when the popup is closed', async () => {
      const auth = setup((p) => (p.popupError = { code: 'auth/popup-closed-by-user' }));
      await auth.linkGoogle();
      expect(auth.secured()).toBeFalse();
      expect(sessionStorage.getItem(LINKING_KEY)).toBeNull();
    });

    it('rethrows a sign-in failure and stays a guest', async () => {
      const auth = setup((p) => (p.popupError = { code: 'auth/network-request-failed' }));
      await expectAsync(auth.linkGoogle()).toBeRejected();
      expect(auth.secured()).toBeFalse();
    });

    it('finishes the link after a redirect sign-in returns', async () => {
      sessionStorage.setItem(LINKING_KEY, '1');
      const auth = setup((p) => (p.user = { uid: 'u1' }));
      await auth.whenReady();

      const req = http.expectOne('/api/me/claim');
      expect(req.request.body).toEqual({ username: 'lemonjoe' });
      req.flush({ id: 1, username: 'lemonjoe' });
      await new Promise((resolve) => setTimeout(resolve));

      expect(auth.secured()).toBeTrue();
    });
  });

  it('apiErrorCode reads the API error code', () => {
    expect(apiErrorCode(new Error('x'))).toBeNull();
  });
});
