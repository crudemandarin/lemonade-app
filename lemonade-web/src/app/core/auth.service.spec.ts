import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { AuthService, IS_STANDALONE, apiErrorCode } from './auth.service';
import { SessionService } from './session.service';
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

  afterEach(() => http?.verify());

  it('is not ready until the stored session has been checked', async () => {
    const auth = setup((p) => (p.holdFirstEvent = true));
    expect(auth.ready()).toBeFalse();

    port.emit(null);
    await auth.whenReady();

    expect(auth.ready()).toBeTrue();
    expect(auth.firebaseUser()).toBeNull();
  });

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

  it('apiErrorCode reads the API error code', () => {
    expect(apiErrorCode(new Error('x'))).toBeNull();
  });
});
