import { Component, input } from '@angular/core';

import { GameStats } from '../../../../core/api.models';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** The running totals of a finished run: peak cash, earnings, building, upkeep and output. */
@Component({
  selector: 'app-run-stats',
  standalone: true,
  imports: [MoneyPipe],
  templateUrl: './run-stats.component.html',
  styleUrl: './run-stats.component.scss',
})
export class RunStatsComponent {
  readonly stats = input.required<GameStats>();
}
