import { TestBed } from '@angular/core/testing';

import { SessionService, USERNAME_KEY } from './session.service';

describe('SessionService', () => {
  beforeEach(() => localStorage.removeItem(USERNAME_KEY));
  afterEach(() => localStorage.removeItem(USERNAME_KEY));

  it('starts signed out when nothing is stored', () => {
    const session = TestBed.inject(SessionService);
    expect(session.username()).toBeNull();
    expect(session.secured()).toBeFalse();
  });

  it('restores a guest from a stored username', () => {
    localStorage.setItem(USERNAME_KEY, 'lemonjoe');
    const session = TestBed.inject(SessionService);
    expect(session.username()).toBe('lemonjoe');
    expect(session.secured()).toBeFalse();
  });

  it('remembers a guest and forgets them on sign out', () => {
    const session = TestBed.inject(SessionService);

    session.signIn('lemonjoe');
    expect(session.username()).toBe('lemonjoe');
    expect(localStorage.getItem(USERNAME_KEY)).toBe('lemonjoe');

    session.signOut();
    expect(session.username()).toBeNull();
    expect(localStorage.getItem(USERNAME_KEY)).toBeNull();
  });

  it('keeps a secured account in memory only, and drops the guest name', () => {
    localStorage.setItem(USERNAME_KEY, 'lemonjoe');
    const session = TestBed.inject(SessionService);

    session.signInSecured('lemonjoe');

    expect(session.username()).toBe('lemonjoe');
    expect(session.secured()).toBeTrue();
    expect(localStorage.getItem(USERNAME_KEY)).toBeNull();
  });
});
