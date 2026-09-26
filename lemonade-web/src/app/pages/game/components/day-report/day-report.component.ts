import { Component, computed, input } from '@angular/core';

import { DayReport } from '../../../../core/api.models';
import { resourceLabel } from '../../../../core/resources';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** The body of one day's report, rendered straight from the server's DayReport. Used by the end-of-day modal and the past-days drawer. */
@Component({
  selector: 'app-day-report',
  standalone: true,
  imports: [IconComponent, MoneyPipe],
  templateUrl: './day-report.component.html',
  styleUrl: './day-report.component.scss',
})
export class DayReportComponent {
  readonly report = input.required<DayReport>();

  protected readonly label = resourceLabel;

  /** Perishables that went off overnight, in the server's order. */
  protected readonly spoiledList = computed(() =>
    Object.entries(this.report().spoiled ?? {})
      .filter(([, cases]) => cases > 0)
      .map(([resource, cases]) => ({ resource, cases })),
  );

  /** Every price, lemonade first, then the rest in the server's order. */
  protected readonly prices = computed(() => {
    const all = this.report().priceChanges;
    return [
      ...all.filter((c) => c.resource === 'lemonade'),
      ...all.filter((c) => c.resource !== 'lemonade'),
    ];
  });
}
