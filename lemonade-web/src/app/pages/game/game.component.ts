import { Component, OnInit, inject } from '@angular/core';

import { GameStore } from '../../core/game.store';
import { IconComponent } from '../../shared/icon/icon.component';
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
    IconComponent,
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

  ngOnInit(): void {
    this.store.load();
  }
}
