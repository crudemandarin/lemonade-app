import { TestBed } from '@angular/core/testing';

import { AuthService } from '../auth.service';
import { AuthUser, FirebaseAuthPort } from '../firebase-auth.port';
import { SessionService } from '../session.service';

/** A scriptable stand-in for the Firebase SDK. Provide it with `provideFakeAuth()`. */
export class FakeAuthPort extends FirebaseAuthPort {
  /** Who is signed in when the app starts. */
  user: AuthUser | null = null;
  /** True keeps the first auth-state event back until `emit()` is called. */
  holdFirstEvent = false;
  popupError: { code: string } | null = null;
  popupCalls = 0;
  redirectCalls = 0;
  signOutCalls = 0;
  tokenRequests: boolean[] = [];
  token: string | null = 'token-1';
  refreshedToken: string | null = 'token-2';
  private listener: ((user: AuthUser | null) => void) | null = null;

  init(onChange: (user: AuthUser | null) => void): void {
    this.listener = onChange;
    if (!this.holdFirstEvent) {
      onChange(this.user);
    }
  }

  emit(user: AuthUser | null): void {
    this.user = user;
    this.listener?.(user);
  }

  async signInWithPopup(): Promise<void> {
    this.popupCalls++;
    if (this.popupError) {
      throw this.popupError;
    }
    this.emit({ uid: 'uid-1' });
  }

  async signInWithRedirect(): Promise<void> {
    this.redirectCalls++;
  }

  async signOut(): Promise<void> {
    this.signOutCalls++;
    this.emit(null);
  }

  async getIdToken(forceRefresh: boolean): Promise<string | null> {
    this.tokenRequests.push(forceRefresh);
    return forceRefresh ? this.refreshedToken : this.token;
  }
}

export function provideFakeAuth(port = new FakeAuthPort()) {
  return [{ provide: FirebaseAuthPort, useValue: port }];
}

/**
 * Puts the app in the signed-in state with a profile. AuthService is created first,
 * because starting it (with nobody signed in) clears the session.
 */
export function signInForTest(username: string): void {
  TestBed.inject(AuthService);
  TestBed.inject(SessionService).signIn(username);
}
