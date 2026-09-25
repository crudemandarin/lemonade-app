import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { unauthorizedInterceptor } from './unauthorized.interceptor';
import { FakeAuthPort, provideFakeAuth, signInForTest } from './testing/fake-auth';

describe('unauthorizedInterceptor', () => {
  let http: HttpClient;
  let mock: HttpTestingController;
  let port: FakeAuthPort;
  let navigate: jasmine.Spy;

  beforeEach(() => {
    port = new FakeAuthPort();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([unauthorizedInterceptor])),
        provideHttpClientTesting(),
        provideFakeAuth(port),
      ],
    });
    http = TestBed.inject(HttpClient);
    mock = TestBed.inject(HttpTestingController);
    navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    signInForTest('lemonjoe');
  });

  afterEach(() => mock.verify());

  function fail(status: number, body: object) {
    http.get('/api/game').subscribe({ error: () => undefined });
    mock.expectOne('/api/game').flush(body, { status, statusText: '' });
  }

  it('signs out of Firebase and routes to /signin on a 401', () => {
    fail(401, { error: 'unauthorized', message: 'x' });

    expect(port.signOutCalls).toBe(1);
    expect(navigate).toHaveBeenCalledWith('/signin');
  });

  it('sends a player with no username to onboarding on 403 profile_required, staying signed in', () => {
    fail(403, { error: 'profile_required', message: 'x' });

    expect(port.signOutCalls).toBe(0);
    expect(navigate).toHaveBeenCalledWith('/signin/username');
  });

  it('leaves the session alone on other errors', () => {
    fail(409, { error: 'insufficient_funds', message: 'x' });

    expect(port.signOutCalls).toBe(0);
    expect(navigate).not.toHaveBeenCalled();
  });
});
