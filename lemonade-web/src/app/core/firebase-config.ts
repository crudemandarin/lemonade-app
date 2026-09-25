import { Injectable } from '@angular/core';

/**
 * Firebase web settings. They are public identifiers, not secrets, and they are
 * read at startup rather than baked into the build, because one image is deployed
 * to different GCP projects (see nginx.conf.template).
 */
export interface FirebaseConfig {
  apiKey: string;
  authDomain: string;
  projectId: string;
  appId: string;
  /** Local development only: talk to the Firebase Auth Emulator. */
  useEmulator?: boolean;
  emulatorUrl?: string;
}

/**
 * Not under /assets: the service worker caches that path, so after a redeploy to
 * another project it could serve a stale config. Nothing in ngsw-config.json matches
 * this path, and nginx sends no-store for it.
 */
export const FIREBASE_CONFIG_URL = '/config/firebase-config.json';

@Injectable({ providedIn: 'root' })
export class FirebaseConfigService {
  /** Null until loaded, or when the config is missing (sign-in is then unavailable). */
  config: FirebaseConfig | null = null;

  async load(): Promise<void> {
    try {
      const res = await fetch(FIREBASE_CONFIG_URL, { cache: 'no-store' });
      const json: unknown = res.ok ? await res.json() : null;
      this.config = isConfig(json) ? json : null;
    } catch {
      this.config = null;
    }
  }
}

function isConfig(json: unknown): json is FirebaseConfig {
  const c = json as Partial<FirebaseConfig> | null;
  return !!c && !!c.apiKey && !!c.projectId && !!c.appId && !!c.authDomain;
}
