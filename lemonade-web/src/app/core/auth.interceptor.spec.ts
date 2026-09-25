import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { authInterceptor } from './auth.interceptor';
import { SessionService, USERNAME_KEY } from './session.service';
import { FakeAuthPort, provideFakeAuth } from './testing/fake-auth';

describe('authInterceptor', () => {
  let http: HttpClient;
  let mock: HttpTestingController;
  let port: FakeAuthPort;

  beforeEach(() => {
    port = new FakeAuthPort();
    port.user = { uid: 'u1' };
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
        provideFakeAuth(port),
      ],
    });
    http = TestBed.inject(HttpClient);
    mock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    mock.verify();
    localStorage.removeItem(USERNAME_KEY);
  });

  // The interceptor waits for the token promise before sending.
  const tick = () => new Promise((resolve) => setTimeout(resolve));
  const unauthorized = { status: 401, statusText: '' };

  it('sends the ID token as a bearer header', async () => {
    http.get('/api/game').subscribe();
    await tick();
    expect(mock.expectOne('/api/game').request.headers.get('Authorization')).toBe('Bearer token-1');
  });

  describe('as a guest', () => {
    beforeEach(() => {
      port.user = null;
      TestBed.inject(SessionService).signIn('lemonjoe');
    });

    it('sends X-Username and no token', async () => {
      http.get('/api/game').subscribe();
      await tick();
      const req = mock.expectOne('/api/game').request;
      expect(req.headers.get('X-Username')).toBe('lemonjoe');
      expect(req.headers.has('Authorization')).toBeFalse();
      expect(port.tokenRequests).toEqual([]);
    });

    it('does not retry a 401', async () => {
      let status = 0;
      http.get('/api/game').subscribe({ error: (e) => (status = e.status) });
      await tick();
      mock.expectOne('/api/game').flush({}, unauthorized);
      expect(status).toBe(401);
    });
  });

  it('prefers the token to a stored guest name once signed in with Google', async () => {
    TestBed.inject(SessionService).signIn('lemonjoe');
    http.get('/api/game').subscribe();
    await tick();
    const req = mock.expectOne('/api/game').request;
    expect(req.headers.get('Authorization')).toBe('Bearer token-1');
    expect(req.headers.has('X-Username')).toBeFalse();
  });

  it('leaves other URLs alone', async () => {
    http.get('/config/firebase-config.json').subscribe();
    await tick();
    expect(
      mock.expectOne('/config/firebase-config.json').request.headers.has('Authorization'),
    ).toBeFalse();
    expect(port.tokenRequests).toEqual([]);
  });

  it('sends no header when signed out', async () => {
    port.token = null;
    http.get('/api/game').subscribe();
    await tick();
    expect(mock.expectOne('/api/game').request.headers.has('Authorization')).toBeFalse();
  });

  it('refreshes the token once on a 401 and retries with it', async () => {
    let result: unknown;
    http.get('/api/game').subscribe((r) => (result = r));
    await tick();
    mock.expectOne('/api/game').flush({}, unauthorized);
    await tick();

    const retry = mock.expectOne('/api/game');
    expect(retry.request.headers.get('Authorization')).toBe('Bearer token-2');
    expect(port.tokenRequests).toEqual([false, true]);
    retry.flush({ ok: true });
    expect(result).toEqual({ ok: true });
  });

  it('gives up after one retry so the error reaches the sign-out handler', async () => {
    let status = 0;
    http.get('/api/game').subscribe({ error: (e) => (status = e.status) });
    await tick();
    mock.expectOne('/api/game').flush({}, unauthorized);
    await tick();
    mock.expectOne('/api/game').flush({}, unauthorized);

    expect(status).toBe(401);
    mock.expectNone('/api/game');
  });

  it('does not retry other errors', async () => {
    let status = 0;
    http.get('/api/game').subscribe({ error: (e) => (status = e.status) });
    await tick();
    mock.expectOne('/api/game').flush({}, { status: 403, statusText: '' });

    expect(status).toBe(403);
    expect(port.tokenRequests).toEqual([false]);
  });

  it('does not retry when there was no token to refresh', async () => {
    port.token = null;
    let status = 0;
    http.get('/api/game').subscribe({ error: (e) => (status = e.status) });
    await tick();
    mock.expectOne('/api/game').flush({}, unauthorized);
    expect(status).toBe(401);
  });
});
