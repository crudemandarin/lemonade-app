import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { SessionService } from './session.service';
import { usernameInterceptor } from './username.interceptor';

describe('usernameInterceptor', () => {
  let http: HttpClient;
  let mock: HttpTestingController;
  let session: SessionService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([usernameInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpClient);
    mock = TestBed.inject(HttpTestingController);
    session = TestBed.inject(SessionService);
    session.signOut();
  });

  afterEach(() => {
    mock.verify();
    session.signOut();
  });

  it('adds X-Username when signed in', () => {
    session.signIn('lemonjoe');
    http.get('/api/game').subscribe();
    expect(mock.expectOne('/api/game').request.headers.get('X-Username')).toBe('lemonjoe');
  });

  it('sends no header when signed out', () => {
    http.get('/api/game').subscribe();
    expect(mock.expectOne('/api/game').request.headers.has('X-Username')).toBeFalse();
  });
});
