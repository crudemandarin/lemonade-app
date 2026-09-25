import { Component, computed, input, output } from '@angular/core';

import { Projection } from '../../../../core/api.models';
import { RESOURCE_LABELS } from '../../../../core/resources';
import { HelpLinkComponent } from '../../../../shared/help/help-link.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** Day, capital, upkeep and the end-of-day projection, plus End day: the page's only primary button. */
@Component({
  selector: 'app-stats-strip',
  standalone: true,
  imports: [HelpLinkComponent, IconComponent, MoneyPipe],
  templateUrl: './stats-strip.component.html',
  styleUrl: './stats-strip.component.scss',
})
export class StatsStripComponent {
  readonly day = input.required<number>();
  readonly capital = input.required<number>();
  readonly upkeepPerDay = input.required<number>();
  readonly projection = input.required<Projection>();
  /** True while a request is in flight or the app is offline. */
  readonly disabled = input(false);
  readonly endDay = output<void>();

  protected readonly limitedBy = computed(() => {
    const limit = this.projection().limitedBy;
    if (limit === '') return '';
    if (limit === 'production') return 'Limited by production capacity';
    if (limit === 'space') return 'Limited by lemonade storage space';
    return `Limited by ${RESOURCE_LABELS[limit].toLowerCase()}`;
  });

  protected readonly reading = computed(() => {
    const p = this.projection();
    const parts = [`End of day makes ${p.lemonadeToProduce} lemonade`];
    if (p.iceToMelt > 0) parts.push(`${p.iceToMelt} ice will melt`);
    return parts.join(', ') + (this.limitedBy() ? `. ${this.limitedBy()}` : '');
  });
}
