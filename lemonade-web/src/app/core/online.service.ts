import { DestroyRef, Injectable, inject, signal } from '@angular/core';

/**
 * Network status from navigator.onLine and the window online/offline events.
 * Gameplay needs the server, so the UI disables every action while offline.
 */
@Injectable({ providedIn: 'root' })
export class OnlineService {
  private readonly _online = signal(navigator.onLine);

  readonly online = this._online.asReadonly();

  constructor() {
    const setOnline = () => this._online.set(true);
    const setOffline = () => this._online.set(false);
    window.addEventListener('online', setOnline);
    window.addEventListener('offline', setOffline);
    inject(DestroyRef).onDestroy(() => {
      window.removeEventListener('online', setOnline);
      window.removeEventListener('offline', setOffline);
    });
  }
}
