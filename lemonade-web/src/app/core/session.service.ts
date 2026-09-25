import { Injectable, signal } from '@angular/core';

/** A guest's username, kept so a reload does not sign them out. */
export const USERNAME_KEY = 'lemonade.username';

/**
 * The current player's username, and whether the account is secured with Google.
 *
 *  - A **guest** plays on a username alone. The name is kept in localStorage and sent as
 *    X-Username (see auth.interceptor.ts). Not secure: it is the brief's "no password" login.
 *  - A **secured** player (Google linked) is identified by their Firebase token. Firebase
 *    keeps that session in IndexedDB, so only the display name is held here, in memory.
 */
@Injectable({ providedIn: 'root' })
export class SessionService {
  private readonly _username = signal<string | null>(read());
  private readonly _secured = signal(false);

  readonly username = this._username.asReadonly();
  readonly secured = this._secured.asReadonly();

  /** Username-only play. Remembered across reloads. */
  signIn(username: string): void {
    this._username.set(username);
    this._secured.set(false);
    write(username);
  }

  /** A Google-secured account: identified by the token, so no name is stored. */
  signInSecured(username: string): void {
    this._username.set(username);
    this._secured.set(true);
    write(null);
  }

  signOut(): void {
    this._username.set(null);
    this._secured.set(false);
    write(null);
  }
}

// Storage can throw (private mode, blocked site data); treat that as signed out.
function read(): string | null {
  try {
    return localStorage.getItem(USERNAME_KEY);
  } catch {
    return null;
  }
}

function write(username: string | null): void {
  try {
    if (username === null) {
      localStorage.removeItem(USERNAME_KEY);
    } else {
      localStorage.setItem(USERNAME_KEY, username);
    }
  } catch {
    // Keep the in-memory session only.
  }
}
