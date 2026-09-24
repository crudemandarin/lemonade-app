import { Injectable, signal } from '@angular/core';

export const USERNAME_KEY = 'lemonade.username';

/**
 * The signed-in username, persisted in localStorage. There is no server session:
 * the username is sent as X-Username on every request (see username.interceptor.ts).
 */
@Injectable({ providedIn: 'root' })
export class SessionService {
  private readonly _username = signal<string | null>(read());

  readonly username = this._username.asReadonly();

  signIn(username: string): void {
    this._username.set(username);
    write(username);
  }

  signOut(): void {
    this._username.set(null);
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
