import { Component, inject, input } from '@angular/core';

import { IconComponent } from '../icon/icon.component';
import { HelpService } from './help.service';

/** Small "?" button that opens the How to play panel at a glossary term. */
@Component({
  selector: 'app-help-link',
  standalone: true,
  imports: [IconComponent],
  template: `<button
    type="button"
    class="help-link"
    [attr.aria-label]="'What is ' + label().toLowerCase() + '?'"
    (click)="help.show(term())"
  >
    <app-icon name="help" [size]="14" />
  </button>`,
  styles: `
    :host {
      display: inline-flex;
      vertical-align: middle;
    }
    .help-link {
      display: inline-flex;
      padding: 2px;
      border: 0;
      border-radius: 50%;
      background: none;
      color: var(--text-muted);
      opacity: 0.7;
      cursor: pointer;
    }
    .help-link:hover,
    .help-link:focus-visible {
      opacity: 1;
      color: var(--accent-strong);
    }
  `,
})
export class HelpLinkComponent {
  protected readonly help = inject(HelpService);
  /** Glossary term id. */
  readonly term = input.required<string>();
  /** Spoken name, e.g. "bid". */
  readonly label = input.required<string>();
}
