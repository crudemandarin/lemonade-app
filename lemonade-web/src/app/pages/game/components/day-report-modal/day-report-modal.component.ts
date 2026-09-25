import { Component, computed, input, output } from '@angular/core';

import { DayReport } from '../../../../core/api.models';
import { RESOURCE_LABELS } from '../../../../core/resources';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** End-of-day summary, rendered straight from the server's DayReport. */
@Component({
  selector: 'app-day-report-modal',
  standalone: true,
  imports: [IconComponent, MoneyPipe],
  templateUrl: './day-report-modal.component.html',
  styleUrl: './day-report-modal.component.scss',
})
export class DayReportModalComponent {
  readonly report = input.required<DayReport>();
  readonly dismiss = output<void>();

  protected readonly labels = RESOURCE_LABELS;

  /** Every price, lemonade first, then the rest in the server's order. */
  protected readonly prices = computed(() => {
    const all = this.report().priceChanges;
    return [
      ...all.filter((c) => c.resource === 'lemonade'),
      ...all.filter((c) => c.resource !== 'lemonade'),
    ];
  });
}
