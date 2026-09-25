import { Injectable, InjectionToken, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { HttpErrorResponse } from '@angular/common/http';

import { ApiService } from './api.service';
import { AuthUser, FirebaseAuthPort } from './firebase-auth.port';
import { SessionService } from './session.service';

/** True in an installed PWA, where popups are unreliable and redirect sign-in is used. */
export const IS_STANDALONE = new InjectionToken<() => boolean>('IS_STANDALONE', {
  providedIn: 'root',
  factory: () => () =>
    typeof matchMedia === 'function' && matchMedia('(display-mode: standalone)').matches,
});

/** The API error code of a failed request, e.g. "username_taken". */
export function apiErrorCode(err: unknown): string | null {
  if (err instanceof HttpErrorResponse && err.error && typeof err.error === 'object') {
    return (err.error as { error?: string }).error ?? null;
  }
  return null;
}

/**
 * Who is signed in. Google sign-in (Firebase) proves the account; the profile (the
 * username other players see) comes from the API. A signed-in account with no
 * profile has to choose a username or claim a legacy one first.
 */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly port = inject(FirebaseAuthPort);
  private readonly api = inject(ApiService);
  private readonly session = inject(SessionService);
  private readonly isStandalone = inject(IS_STANDALONE);

  private readonly _user = signal<AuthUser | null>(null);
  private readonly _ready = signal(false);
  private profileRequest: Promise<boolean> | null = null;
  private readonly readyPromise: Promise<void>;

  readonly firebaseUser = this._user.asReadonly();
  /** True after the stored session has been checked (avoids a sign-in page flash). */
  readonly ready = this._ready.asReadonly();
  readonly profile = computed(() => {
    const username = this.session.username();
    return username ? { username } : null;
  });

  constructor() {
    this.readyPromise = new Promise((resolve) =>
      this.port.init((user) => {
        this._user.set(user);
        if (user) {
          this.loadProfile().catch(() => undefined);
        } else {
          this.profileRequest = null;
          this.session.signOut();
        }
        this._ready.set(true);
        resolve();
      }),
    );
  }

  whenReady(): Promise<void> {
    return this.readyPromise;
  }

  /** Popup sign-in, falling back to a redirect where popups do not work. */
  async signInWithGoogle(): Promise<void> {
    if (this.isStandalone()) {
      return this.port.signInWithRedirect();
    }
    try {
      await this.port.signInWithPopup();
    } catch (err) {
      switch ((err as { code?: string }).code) {
        case 'auth/popup-blocked':
          return this.port.signInWithRedirect();
        case 'auth/popup-closed-by-user':
        case 'auth/cancelled-popup-request':
          return; // the player changed their mind
        default:
          throw err;
      }
    }
  }

  async signOut(): Promise<void> {
    this.profileRequest = null;
    this.session.signOut();
    await this.port.signOut();
  }

  /** The SDK caches the token and refreshes it before it expires; the app never stores one. */
  getIdToken(forceRefresh = false): Promise<string | null> {
    return this.port.getIdToken(forceRefresh);
  }

  /** Loads the profile once per sign-in. Resolves false when the account has none yet. */
  loadProfile(): Promise<boolean> {
    if (this.session.username()) {
      return Promise.resolve(true);
    }
    this.profileRequest ??= this.fetchProfile();
    return this.profileRequest;
  }

  createProfile(username: string): Promise<void> {
    return this.adopt(this.api.createProfile(username));
  }

  claim(username: string): Promise<void> {
    return this.adopt(this.api.claim(username));
  }

  private async adopt(request: ReturnType<ApiService['me']>): Promise<void> {
    const user = await firstValueFrom(request);
    this.session.signIn(user.username);
  }

  private async fetchProfile(): Promise<boolean> {
    try {
      const user = await firstValueFrom(this.api.me());
      this.session.signIn(user.username);
      return true;
    } catch (err) {
      this.profileRequest = null;
      if (apiErrorCode(err) === 'profile_required') {
        return false;
      }
      throw err;
    }
  }
}
