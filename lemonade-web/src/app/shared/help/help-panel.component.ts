import { Component, ElementRef, HostListener, computed, effect, inject } from '@angular/core';

import { IconComponent } from '../icon/icon.component';
import { HELP_EVENTS, HELP_SECTIONS, HELP_TERMS, QUICK_START } from './glossary';
import { HelpService } from './help.service';

/** Collapsible glossary at the top of the game page. */
@Component({
  selector: 'app-help-panel',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './help-panel.component.html',
  styleUrl: './help-panel.component.scss',
})
export class HelpPanelComponent {
  protected readonly help = inject(HelpService);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  protected readonly sections = HELP_SECTIONS;
  protected readonly events = HELP_EVENTS;
  protected readonly steps = QUICK_START;
  protected readonly terms = computed(() =>
    HELP_TERMS.filter((t) => t.section === this.help.section()),
  );

  constructor() {
    effect(() => {
      const id = this.help.target();
      if (!id) return;
      // Wait for the panel and tab to render before scrolling to the term.
      setTimeout(() => {
        document
          .getElementById('help-' + id)
          ?.scrollIntoView({ block: 'center', behavior: 'smooth' });
      });
      setTimeout(() => this.help.target.set(null), 2000);
    });
  }

  @HostListener('document:keydown.escape')
  protected close(): void {
    this.help.open.set(false);
  }

  /** Clicking anywhere else closes the menu; "?" links re-open it themselves. */
  @HostListener('document:click', ['$event'])
  protected onDocumentClick(event: MouseEvent): void {
    const target = event.target as Element;
    if (!this.host.nativeElement.contains(target) && !target.closest('.help-link')) {
      this.help.open.set(false);
    }
  }
}
