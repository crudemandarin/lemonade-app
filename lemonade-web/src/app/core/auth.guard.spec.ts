import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree } from '@angular/router';

import { authGuard } from './auth.guard';
import { SessionService } from './session.service';

describe('authGuard', () => {
  let session: SessionService;

  const run = () =>
    TestBed.runInInjectionContext(() =>
      authGuard({} as ActivatedRouteSnapshot, {} as RouterStateSnapshot),
    );

  beforeEach(() => {
    session = TestBed.inject(SessionService);
    session.signOut();
  });

  afterEach(() => session.signOut());

  it('allows a signed-in user', () => {
    session.signIn('lemonjoe');
    expect(run()).toBeTrue();
  });

  it('redirects to /signin when signed out', () => {
    const result = run() as UrlTree;
    expect(TestBed.inject(Router).serializeUrl(result)).toBe('/signin');
  });
});
