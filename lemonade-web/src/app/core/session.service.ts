import { Injectable, signal } from '@angular/core';

/** The pre-Firebase login stored the username here. It is no longer read. */
export const LEGACY_USERNAME_KEY = 'lemonade.username';

/**
 * The signed-in player's username: the public profile, held in memory. AuthService
 * fills it from `GET /api/me` and clears it on sign-out; Firebase persists the actual
 * session, so nothing is stored here.
 */
@Injectable({ providedIn: 'root' })
export class SessionService {
  private readonly _username = signal<string | null>(null);

  readonly username = this._username.asReadonly();

  constructor() {
    try {
      localStorage.removeItem(LEGACY_USERNAME_KEY);
    } catch {
      // Storage can throw (private mode); there is nothing to clear then.
    }
  }

  signIn(username: string): void {
    this._username.set(username);
  }

  signOut(): void {
    this._username.set(null);
  }
}
