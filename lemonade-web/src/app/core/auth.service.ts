import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, InjectionToken, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { ApiService } from './api.service';
import { AuthUser, FirebaseAuthPort } from './firebase-auth.port';
import { FirebaseConfigService } from './firebase-config';
import { SessionService } from './session.service';

/** True in an installed PWA, where popups are unreliable and redirect sign-in is used. */
export const IS_STANDALONE = new InjectionToken<() => boolean>('IS_STANDALONE', {
  providedIn: 'root',
  factory: () => () =>
    typeof matchMedia === 'function' && matchMedia('(display-mode: standalone)').matches,
});

/**
 * How long startup waits for Firebase to report the stored session. It always does in
 * practice; in one observed case (a brand-new browser profile whose first page load
 * was abandoned mid-start) the next load's IndexedDB stayed blocked, and without this
 * the app would sit on "Loading…" for good.
 */
export const READY_TIMEOUT_MS = 10_000;

/** Set while a guest is linking Google, so a redirect sign-in can finish the job on return. */
export const LINKING_KEY = 'lemonade.linking';

/** The API error code of a failed request, e.g. "username_taken". */
export function apiErrorCode(err: unknown): string | null {
  if (err instanceof HttpErrorResponse && err.error && typeof err.error === 'object') {
    return (err.error as { error?: string }).error ?? null;
  }
  return null;
}

/**
 * Who is playing. Everyone can play as a guest on a username alone. Signing in with
 * Google (Firebase) is optional: it secures an account, after which only the Google
 * account can play it. The username is the only public identity either way.
 */
@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly port = inject(FirebaseAuthPort);
  private readonly api = inject(ApiService);
  private readonly session = inject(SessionService);
  private readonly isStandalone = inject(IS_STANDALONE);
  private readonly config = inject(FirebaseConfigService);

  private readonly _user = signal<AuthUser | null>(null);
  private readonly _ready = signal(false);
  private readonly _linkError = signal<string | null>(null);
  private profileRequest: Promise<boolean> | null = null;
  private readonly readyPromise: Promise<void>;

  readonly firebaseUser = this._user.asReadonly();
  /** True after the stored session has been checked (avoids a sign-in page flash). */
  readonly ready = this._ready.asReadonly();
  readonly profile = computed(() => {
    const username = this.session.username();
    return username ? { username } : null;
  });
  /** The account is protected by Google, not just by knowing its name. */
  readonly secured = this.session.secured;
  /** Why the last attempt to link Google failed (an error code), if it did. */
  readonly linkError = this._linkError.asReadonly();

  /** Google sign-in is off when the deployment has no Firebase settings. */
  get googleAvailable(): boolean {
    return this.config.config !== null;
  }

  constructor() {
    this.readyPromise = new Promise((resolve) => {
      const markReady = () => {
        this._ready.set(true);
        resolve();
      };
      const timer = setTimeout(markReady, READY_TIMEOUT_MS);
      this.port.init((user) => {
        clearTimeout(timer);
        this._user.set(user);
        if (user) {
          if (isLinking() && this.session.username()) {
            // A guest is securing their account: the account has no profile until the link
            // lands, so do not probe for one. Only a redirect return (the first event) has
            // to finish the link here; a popup's event is handled by linkGoogle itself.
            if (!this._ready()) {
              this.completeLink().catch(() => undefined);
            }
          } else {
            this.loadProfile().catch(() => undefined);
          }
        } else {
          this.profileRequest = null;
          if (this.session.secured()) {
            this.session.signOut(); // a guest's stored name is not Firebase's to clear
          }
        }
        markReady();
      });
    });
  }

  whenReady(): Promise<void> {
    return this.readyPromise;
  }

  /** Username-only play: starts or resumes a guest. Rejects `account_secured` for a protected name. */
  async guestSignIn(username: string): Promise<string> {
    const user = await firstValueFrom(this.api.login(username));
    this.session.signIn(user.username);
    return user.username;
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

  /**
   * Secures the current guest account: signs in with Google and links the username to it.
   * On failure the player stays a guest. A redirect sign-in leaves the page; the link is
   * finished on return (see the constructor).
   */
  async linkGoogle(): Promise<void> {
    this._linkError.set(null);
    setLinking(true);
    try {
      await this.signInWithGoogle();
    } catch (err) {
      setLinking(false);
      throw err;
    }
    if (!this.firebaseUser()) {
      setLinking(false); // popup closed, or a redirect is under way
      return;
    }
    await this.completeLink();
  }

  private async completeLink(): Promise<void> {
    const username = this.session.username();
    setLinking(false);
    if (!username) {
      return;
    }
    try {
      const user = await firstValueFrom(this.api.claim(username));
      this.session.signInSecured(user.username);
    } catch (err) {
      // Stay a guest: drop the Google sign-in that could not be linked.
      await this.port.signOut();
      this._linkError.set(apiErrorCode(err) ?? 'failed');
      throw err;
    }
  }

  /** Signs out of everything: a guest's stored name and any Google session. */
  async signOut(): Promise<void> {
    this.profileRequest = null;
    this.session.signOut();
    await this.port.signOut();
  }

  /** Forgets a guest name without touching Google (its account was secured elsewhere). */
  forgetGuest(): void {
    this.session.signOut();
  }

  /** The SDK caches the token and refreshes it before it expires; the app never stores one. */
  getIdToken(forceRefresh = false): Promise<string | null> {
    return this.port.getIdToken(forceRefresh);
  }

  /** Loads the profile once per sign-in. Resolves false when the account has none yet. */
  loadProfile(): Promise<boolean> {
    if (this.session.secured()) {
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
    this.session.signInSecured(user.username);
  }

  private async fetchProfile(): Promise<boolean> {
    try {
      const user = await firstValueFrom(this.api.me());
      this.session.signInSecured(user.username);
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

function isLinking(): boolean {
  try {
    return sessionStorage.getItem(LINKING_KEY) === '1';
  } catch {
    return false;
  }
}

function setLinking(on: boolean): void {
  try {
    if (on) {
      sessionStorage.setItem(LINKING_KEY, '1');
    } else {
      sessionStorage.removeItem(LINKING_KEY);
    }
  } catch {
    // Without sessionStorage a redirect link cannot resume; the popup path still works.
  }
}
