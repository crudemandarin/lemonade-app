import { Injectable, signal } from '@angular/core';

import { UnlockedAchievement } from './api.models';

export interface Toast {
  id: number;
  text: string;
}

/** How long a toast stays, in milliseconds. */
export const TOAST_MS = 5000;

/**
 * Short, stacked notices. One line each, no emoji, dismissed on their own after
 * TOAST_MS or with the close button. The toast host announces them via aria-live.
 */
@Injectable({ providedIn: 'root' })
export class ToastService {
  private nextId = 1;
  private readonly _toasts = signal<Toast[]>([]);
  readonly toasts = this._toasts.asReadonly();

  show(text: string): void {
    const id = this.nextId++;
    this._toasts.update((list) => [...list, { id, text }]);
    setTimeout(() => this.dismiss(id), TOAST_MS);
  }

  /** One toast per achievement the server says a mutation just unlocked. */
  unlocked(list: UnlockedAchievement[] | undefined): void {
    for (const a of list ?? []) {
      this.show(`Achievement unlocked: ${a.name}`);
    }
  }

  dismiss(id: number): void {
    this._toasts.update((list) => list.filter((t) => t.id !== id));
  }
}
