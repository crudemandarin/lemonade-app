import { Component, OnInit, inject, signal } from '@angular/core';

import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';
import { CardComponent } from '../../shared/card/card.component';
import { ConfirmDialogComponent } from '../../shared/confirm-dialog/confirm-dialog.component';
import { HelpPanelComponent } from '../../shared/help/help-panel.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { MoneyPipe } from '../../shared/money.pipe';
import { TimelineChartsComponent } from '../../shared/timeline-charts/timeline-charts.component';
import { DayReportModalComponent } from './components/day-report-modal/day-report-modal.component';
import { EventsBannerComponent } from './components/events-banner/events-banner.component';
import { FacilitiesPanelComponent } from './components/facilities-panel/facilities-panel.component';
import { GameOverComponent } from './components/game-over/game-over.component';
import { MarketPanelComponent } from './components/market-panel/market-panel.component';
import { StatsStripComponent } from './components/stats-strip/stats-strip.component';

/** Wires the presentational game components to the GameStore. */
@Component({
  selector: 'app-game',
  standalone: true,
  imports: [
    CardComponent,
    ConfirmDialogComponent,
    IconComponent,
    MoneyPipe,
    HelpPanelComponent,
    TimelineChartsComponent,
    StatsStripComponent,
    EventsBannerComponent,
    MarketPanelComponent,
    FacilitiesPanelComponent,
    DayReportModalComponent,
    GameOverComponent,
  ],
  templateUrl: './game.component.html',
  styleUrl: './game.component.scss',
})
export class GameComponent implements OnInit {
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;

  protected readonly confirmingGiveUp = signal(false);

  ngOnInit(): void {
    this.store.load();
  }

  protected giveUp(): void {
    this.confirmingGiveUp.set(false);
    this.store.giveUp();
  }
}
