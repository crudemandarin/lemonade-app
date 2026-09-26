import { RunNavComponent } from '../../shared/run-nav/run-nav.component';
import { Component, OnInit, computed, inject, signal } from '@angular/core';

import { resourceLabel } from '../../core/resources';
import { DayReport, ReportSummary, Resource } from '../../core/api.models';
import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';
import { CardComponent } from '../../shared/card/card.component';
import { ConfirmDialogComponent } from '../../shared/confirm-dialog/confirm-dialog.component';
import { IconComponent } from '../../shared/icon/icon.component';
import { MoneyPipe } from '../../shared/money.pipe';
import { TimelineChartsComponent } from '../../shared/timeline-charts/timeline-charts.component';
import { DayReportModalComponent } from './components/day-report-modal/day-report-modal.component';
import { EventsBannerComponent } from './components/events-banner/events-banner.component';
import { FacilitiesPanelComponent } from './components/facilities-panel/facilities-panel.component';
import { GameOverComponent } from './components/game-over/game-over.component';
import { PastDaysDrawerComponent } from './components/past-days-drawer/past-days-drawer.component';
import { MarketPanelComponent } from './components/market-panel/market-panel.component';
import { EmpireStripComponent } from './components/empire-strip/empire-strip.component';
import { VictoryBannerComponent } from './components/victory-banner/victory-banner.component';
import { StatsStripComponent } from './components/stats-strip/stats-strip.component';

const TIMELINE_KEY = 'lemonade.timelineOpen';

function loadTimelineOpen(): boolean {
  try {
    return localStorage.getItem(TIMELINE_KEY) !== 'closed';
  } catch {
    return true;
  }
}

const ALERTS_KEY = 'lemonade.priceAlerts';

function loadAlerts(): Record<Resource, number> {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(ALERTS_KEY) ?? '{}');
    return parsed && typeof parsed === 'object' ? (parsed as Record<Resource, number>) : {};
  } catch {
    return {};
  }
}

/** Wires the presentational game components to the GameStore. */
@Component({
  selector: 'app-game',
  standalone: true,
  imports: [
    RunNavComponent,
    CardComponent,
    ConfirmDialogComponent,
    IconComponent,
    MoneyPipe,
    TimelineChartsComponent,
    EmpireStripComponent,
    StatsStripComponent,
    VictoryBannerComponent,
    EventsBannerComponent,
    MarketPanelComponent,
    FacilitiesPanelComponent,
    DayReportModalComponent,
    GameOverComponent,
    PastDaysDrawerComponent,
  ],
  templateUrl: './game.component.html',
  styleUrl: './game.component.scss',
})
export class GameComponent implements OnInit {
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;

  protected readonly confirmingGiveUp = signal(false);

  /** The timeline card can be folded away; the choice is remembered. */
  protected readonly timelineOpen = signal(loadTimelineOpen());

  /** Price alert levels (the Price alerts upgrade), kept in this browser. */
  protected readonly alerts = signal<Record<Resource, number>>(loadAlerts());

  /** Yesterday's trades, in order, for the Order book upgrade's "repeat" button. */
  /** Only commodities the player has a use for, so the market does not open with 21 rows. */
  protected readonly market = computed(
    () => this.store.game()?.resources.filter((r) => r.unlocked) ?? [],
  );

  protected readonly unlockedKeys = computed(() => this.market().map((r) => r.resource));

  /** Perishables that will go off tonight, as "5 Limes" phrases; empty when nothing will. */
  protected readonly spoilWarning = computed(() => {
    const will = this.store.game()?.projection.willSpoil ?? {};
    return Object.entries(will)
      .filter(([, cases]) => cases > 0)
      .map(([r, cases]) => `${cases} ${resourceLabel(r).toLowerCase()}`);
  });

  protected readonly yesterdaysTrades = computed(() => {
    const game = this.store.game();
    if (!game || !game.features.includes('repeat_trades')) return [];
    return game.timeline.filter(
      (p) => (p.kind === 'buy' || p.kind === 'sell') && p.day === game.day - 1 && p.resource,
    );
  });

  protected readonly pastOpen = signal(false);
  protected readonly pastDays = signal<ReportSummary[] | null>(null);
  protected readonly pastSelected = signal<number | null>(null);
  protected readonly pastReport = signal<DayReport | null>(null);
  protected readonly pastFailed = signal(false);
  /** Full reports already fetched, so stepping back and forth costs nothing. */
  private readonly pastCache = new Map<number, DayReport>();

  ngOnInit(): void {
    this.store.load();
  }

  protected toggleTimeline(): void {
    const open = !this.timelineOpen();
    this.timelineOpen.set(open);
    try {
      localStorage.setItem(TIMELINE_KEY, open ? 'open' : 'closed');
    } catch {
      // Storage can be blocked; the choice just won't persist.
    }
  }

  protected setAlert(change: { resource: Resource; level: number | null }): void {
    const next = { ...this.alerts() };
    if (change.level === null) {
      delete next[change.resource];
    } else {
      next[change.resource] = change.level;
    }
    this.alerts.set(next);
    try {
      localStorage.setItem(ALERTS_KEY, JSON.stringify(next));
    } catch {
      // Storage can be blocked; the alert just won't persist.
    }
  }

  /** Repeats yesterday's trades one by one, as many cases as cash, space and stock allow. */
  protected async repeatTrades(): Promise<void> {
    for (const p of this.yesterdaysTrades()) {
      const resource = p.resource as Resource;
      if (p.kind === 'buy') {
        await this.store.buy(resource, p.qty, true);
      } else {
        await this.store.sell(resource, p.qty, true);
      }
    }
  }

  protected giveUp(): void {
    this.confirmingGiveUp.set(false);
    this.store.giveUp();
  }

  /** Opens the drawer on the most recent ended day. */
  protected async openPastDays(): Promise<void> {
    this.pastOpen.set(true);
    this.pastFailed.set(false);
    this.pastDays.set(null);
    this.pastReport.set(null);
    this.pastCache.clear();
    try {
      const days = await this.store.reportSummaries();
      this.pastDays.set(days);
      if (days.length > 0) {
        await this.selectPastDay(days[days.length - 1].day);
      }
    } catch {
      this.pastFailed.set(true);
    }
  }

  protected async selectPastDay(day: number): Promise<void> {
    this.pastSelected.set(day);
    const cached = this.pastCache.get(day);
    if (cached) {
      this.pastReport.set(cached);
      return;
    }
    this.pastReport.set(null);
    try {
      const report = await this.store.reportForDay(day);
      this.pastCache.set(day, report);
      if (this.pastSelected() === day) {
        this.pastReport.set(report);
      }
    } catch {
      this.pastFailed.set(true);
    }
  }
}
