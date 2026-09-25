import { TestBed } from '@angular/core/testing';

import { LEGACY_USERNAME_KEY, SessionService } from './session.service';

describe('SessionService', () => {
  afterEach(() => localStorage.removeItem(LEGACY_USERNAME_KEY));

  it('starts signed out', () => {
    expect(TestBed.inject(SessionService).username()).toBeNull();
  });

  it('holds the username in memory only', () => {
    const session = TestBed.inject(SessionService);

    session.signIn('lemonjoe');
    expect(session.username()).toBe('lemonjoe');
    expect(localStorage.getItem(LEGACY_USERNAME_KEY)).toBeNull();

    session.signOut();
    expect(session.username()).toBeNull();
  });

  it('ignores and clears the old localStorage username', () => {
    localStorage.setItem(LEGACY_USERNAME_KEY, 'lemonjoe');
    expect(TestBed.inject(SessionService).username()).toBeNull();
    expect(localStorage.getItem(LEGACY_USERNAME_KEY)).toBeNull();
  });
});
