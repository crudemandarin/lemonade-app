import { Injectable, signal } from '@angular/core';

import { HELP_TERMS, HelpSection } from './glossary';

/** State of the How to play panel, shared so any "?" link can open it at a term. */
@Injectable({ providedIn: 'root' })
export class HelpService {
  readonly open = signal(false);
  readonly section = signal<HelpSection>('quickstart');
  /** Term to scroll to and highlight; cleared shortly after. */
  readonly target = signal<string | null>(null);

  toggle(): void {
    this.open.update((o) => !o);
  }

  show(id: string): void {
    const term = HELP_TERMS.find((t) => t.id === id);
    if (!term) return;
    this.section.set(term.section);
    this.open.set(true);
    this.target.set(id);
  }
}
