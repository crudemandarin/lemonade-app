import { Component, computed, effect, inject } from '@angular/core';

import { CardComponent } from '../card/card.component';
import { IconComponent } from '../icon/icon.component';
import { HELP_EVENTS, HELP_SECTIONS, HELP_TERMS, QUICK_START } from './glossary';
import { HelpService } from './help.service';

/** Collapsible glossary at the top of the game page. */
@Component({
  selector: 'app-help-panel',
  standalone: true,
  imports: [CardComponent, IconComponent],
  templateUrl: './help-panel.component.html',
  styleUrl: './help-panel.component.scss',
})
export class HelpPanelComponent {
  protected readonly help = inject(HelpService);
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
}
