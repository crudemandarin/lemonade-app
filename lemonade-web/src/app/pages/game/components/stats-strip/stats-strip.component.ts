import { Component, input, output } from '@angular/core';

import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** Day, capital, and upkeep, plus End day: the page's only primary button. */
@Component({
  selector: 'app-stats-strip',
  standalone: true,
  imports: [IconComponent, MoneyPipe],
  templateUrl: './stats-strip.component.html',
  styleUrl: './stats-strip.component.scss',
})
export class StatsStripComponent {
  readonly day = input.required<number>();
  readonly capital = input.required<number>();
  readonly upkeepPerDay = input.required<number>();
  /** True while a request is in flight or the app is offline. */
  readonly disabled = input(false);
  readonly endDay = output<void>();
}
