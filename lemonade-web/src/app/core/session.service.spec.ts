import { TestBed } from '@angular/core/testing';

import { SessionService, USERNAME_KEY } from './session.service';

describe('SessionService', () => {
  beforeEach(() => localStorage.removeItem(USERNAME_KEY));
  afterEach(() => localStorage.removeItem(USERNAME_KEY));

  it('starts signed out when nothing is stored', () => {
    expect(TestBed.inject(SessionService).username()).toBeNull();
  });

  it('restores a stored username', () => {
    localStorage.setItem(USERNAME_KEY, 'lemonjoe');
    expect(TestBed.inject(SessionService).username()).toBe('lemonjoe');
  });

  it('persists sign in and clears on sign out', () => {
    const session = TestBed.inject(SessionService);

    session.signIn('lemonjoe');
    expect(session.username()).toBe('lemonjoe');
    expect(localStorage.getItem(USERNAME_KEY)).toBe('lemonjoe');

    session.signOut();
    expect(session.username()).toBeNull();
    expect(localStorage.getItem(USERNAME_KEY)).toBeNull();
  });
});
