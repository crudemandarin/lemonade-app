import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { SessionService } from './session.service';
import { unauthorizedInterceptor } from './unauthorized.interceptor';

describe('unauthorizedInterceptor', () => {
  let http: HttpClient;
  let mock: HttpTestingController;
  let session: SessionService;
  let navigate: jasmine.Spy;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        provideHttpClient(withInterceptors([unauthorizedInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpClient);
    mock = TestBed.inject(HttpTestingController);
    session = TestBed.inject(SessionService);
    navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    session.signIn('lemonjoe');
  });

  afterEach(() => {
    mock.verify();
    session.signOut();
  });

  it('signs out and routes to /signin on a 401 from the game API', () => {
    http.get('/api/game').subscribe({ error: () => undefined });
    mock
      .expectOne('/api/game')
      .flush({ error: 'unauthorized', message: 'x' }, { status: 401, statusText: '' });

    expect(session.username()).toBeNull();
    expect(navigate).toHaveBeenCalledWith('/signin');
  });

  it('leaves the session alone on other errors', () => {
    http.get('/api/game').subscribe({ error: () => undefined });
    mock
      .expectOne('/api/game')
      .flush({ error: 'insufficient_funds', message: 'x' }, { status: 409, statusText: '' });

    expect(session.username()).toBe('lemonjoe');
    expect(navigate).not.toHaveBeenCalled();
  });
});
