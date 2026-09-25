import { Component, input, output } from '@angular/core';

import {
  GameStats,
  GameStatus,
  NetWorth,
  PricePoint,
  TimelinePoint,
} from '../../../../core/api.models';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';
import { TimelineChartsComponent } from '../../../../shared/timeline-charts/timeline-charts.component';

/** The result screen (bankrupt or gave up): how it ended, the final score, a summary of the whole game, and its history charts. */
@Component({
  selector: 'app-game-over',
  standalone: true,
  imports: [CardComponent, IconComponent, MoneyPipe, TimelineChartsComponent],
  templateUrl: './game-over.component.html',
  styleUrl: './game-over.component.scss',
})
export class GameOverComponent {
  /** How the run ended. */
  readonly status = input<Exclude<GameStatus, 'active'>>('bankrupt');
  readonly netWorth = input<NetWorth | null>(null);
  readonly day = input.required<number>();
  readonly capital = input.required<number>();
  readonly stats = input.required<GameStats>();
  readonly timeline = input.required<TimelinePoint[]>();
  readonly priceLog = input<PricePoint[]>([]);
  readonly basePrices = input<number[]>([]);
  readonly disabled = input(false);
  readonly newGame = output<void>();
}
