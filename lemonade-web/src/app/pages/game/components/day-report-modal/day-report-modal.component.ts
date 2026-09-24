import { Component, computed, input, output } from '@angular/core';

import { DayReport } from '../../../../core/api.models';
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

  protected readonly lemonadePrice = computed(() =>
    this.report().priceChanges.find((c) => c.resource === 'lemonade'),
  );
}
