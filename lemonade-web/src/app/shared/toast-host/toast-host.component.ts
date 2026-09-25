import { Component, inject } from '@angular/core';

import { ToastService } from '../../core/toast.service';

/** Renders the toast stack in a polite live region, so screen readers hear each unlock. */
@Component({
  selector: 'app-toast-host',
  standalone: true,
  template: `
    <div class="stack" role="status" aria-live="polite" aria-atomic="false">
      @for (t of toasts.toasts(); track t.id) {
        <p class="toast">
          <span>{{ t.text }}</span>
          <button
            type="button"
            class="close"
            aria-label="Dismiss notification"
            (click)="toasts.dismiss(t.id)"
          >
            Dismiss
          </button>
        </p>
      }
    </div>
  `,
  styles: `
    .stack {
      position: fixed;
      right: 0.75rem;
      bottom: 0.75rem;
      left: 0.75rem;
      z-index: 50;
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
      align-items: flex-end;
      pointer-events: none;
    }
    .toast {
      display: flex;
      gap: 0.75rem;
      align-items: center;
      justify-content: space-between;
      box-sizing: border-box;
      max-width: 100%;
      margin: 0;
      padding: 0.5rem 0.75rem;
      border: 1px solid var(--border);
      border-radius: 8px;
      background: var(--surface, #fff);
      color: var(--text);
      font-size: 0.875rem;
      box-shadow: 0 2px 8px rgb(0 0 0 / 0.15);
      pointer-events: auto;
    }
    .close {
      flex: none;
      border: 0;
      background: none;
      color: var(--text-muted);
      font: inherit;
      font-size: 0.75rem;
      text-decoration: underline;
      cursor: pointer;
    }
  `,
})
export class ToastHostComponent {
  protected readonly toasts = inject(ToastService);
}
