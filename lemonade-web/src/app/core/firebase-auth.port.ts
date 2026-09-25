import { Injectable, inject } from '@angular/core';
import { FirebaseError, initializeApp } from 'firebase/app';
import {
  Auth,
  GoogleAuthProvider,
  connectAuthEmulator,
  getAuth,
  getRedirectResult,
  onAuthStateChanged,
  signInWithPopup,
  signInWithRedirect,
  signOut,
} from 'firebase/auth';

import { FirebaseConfigService } from './firebase-config';

export interface AuthUser {
  uid: string;
}

/**
 * The few Firebase calls the app needs. AuthService depends on this, not on the SDK,
 * so specs can swap in a fake.
 */
@Injectable({
  providedIn: 'root',
  useFactory: () => new FirebaseSdkPort(inject(FirebaseConfigService)),
})
export abstract class FirebaseAuthPort {
  /** Subscribes to sign-in changes. The first call means the stored session is resolved. */
  abstract init(onChange: (user: AuthUser | null) => void): void;
  abstract signInWithPopup(): Promise<void>;
  abstract signInWithRedirect(): Promise<void>;
  abstract signOut(): Promise<void>;
  /** A cached token, refreshed by the SDK when near expiry. Null when signed out. */
  abstract getIdToken(forceRefresh: boolean): Promise<string | null>;
}

/** The real SDK. Firebase keeps the session in IndexedDB; tokens are never stored by the app. */
export class FirebaseSdkPort extends FirebaseAuthPort {
  private auth: Auth | null = null;

  constructor(private readonly settings: FirebaseConfigService) {
    super();
  }

  init(onChange: (user: AuthUser | null) => void): void {
    const cfg = this.settings.config;
    if (!cfg) {
      onChange(null);
      return;
    }
    const auth = getAuth(
      initializeApp({
        apiKey: cfg.apiKey,
        authDomain: cfg.authDomain,
        projectId: cfg.projectId,
        appId: cfg.appId,
      }),
    );
    if (cfg.useEmulator) {
      connectAuthEmulator(auth, cfg.emulatorUrl ?? 'http://localhost:9099', {
        disableWarnings: true,
      });
    }
    this.auth = auth;
    // Completes a redirect sign-in; the outcome arrives through onAuthStateChanged.
    getRedirectResult(auth).catch(() => undefined);
    onAuthStateChanged(auth, (user) => onChange(user ? { uid: user.uid } : null));
  }

  async signInWithPopup(): Promise<void> {
    await signInWithPopup(this.need(), new GoogleAuthProvider());
  }

  async signInWithRedirect(): Promise<void> {
    await signInWithRedirect(this.need(), new GoogleAuthProvider());
  }

  async signOut(): Promise<void> {
    if (this.auth) {
      await signOut(this.auth);
    }
  }

  async getIdToken(forceRefresh: boolean): Promise<string | null> {
    return (await this.auth?.currentUser?.getIdToken(forceRefresh)) ?? null;
  }

  private need(): Auth {
    if (!this.auth) {
      throw new FirebaseError('auth/unavailable', 'Sign-in is not configured.');
    }
    return this.auth;
  }
}
