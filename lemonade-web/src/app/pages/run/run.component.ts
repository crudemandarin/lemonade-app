import { DatePipe } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';

import { DayReport, RunDetail } from '../../core/api.models';
import { ScoresService } from '../../core/scores.service';
import { CardComponent } from '../../shared/card/card.component';
import { MoneyPipe } from '../../shared/money.pipe';
import { TimelineChartsComponent } from '../../shared/timeline-charts/timeline-charts.component';
import { ReportStepperComponent } from '../game/components/report-stepper/report-stepper.component';
import { RunStatsComponent } from '../game/components/run-stats/run-stats.component';

/**
 * One finished run of the signed-in player, in full: the score and how the run ended,
 * the totals, all three charts, and every ended day's report. Only the owner can open
 * it; the server answers 404 for anyone else.
 */
@Component({
  selector: 'app-run',
  standalone: true,
  imports: [
    CardComponent,
    DatePipe,
    MoneyPipe,
    ReportStepperComponent,
    RouterLink,
    RunStatsComponent,
    TimelineChartsComponent,
  ],
  templateUrl: './run.component.html',
  styleUrl: './run.component.scss',
})
export class RunComponent implements OnInit {
  private readonly scores = inject(ScoresService);
  private readonly runId = inject(ActivatedRoute).snapshot.paramMap.get('id') ?? '';

  protected readonly state = signal<'loading' | 'ready' | 'missing' | 'error'>('loading');
  protected readonly run = signal<RunDetail | null>(null);

  protected readonly selectedDay = signal<number | null>(null);
  protected readonly dayReport = signal<DayReport | null>(null);
  protected readonly reportFailed = signal(false);
  private readonly cache = new Map<number, DayReport>();

  ngOnInit(): void {
    this.load();
  }

  protected async load(): Promise<void> {
    this.state.set('loading');
    try {
      const run = await this.scores.run(this.runId);
      this.run.set(run);
      this.state.set('ready');
      if (run.reports.length > 0) {
        await this.selectDay(run.reports[run.reports.length - 1].day);
      }
    } catch (err) {
      const status = (err as { status?: number }).status;
      this.state.set(status === 404 ? 'missing' : 'error');
    }
  }

  protected async selectDay(day: number): Promise<void> {
    this.selectedDay.set(day);
    this.reportFailed.set(false);
    const cached = this.cache.get(day);
    if (cached) {
      this.dayReport.set(cached);
      return;
    }
    this.dayReport.set(null);
    try {
      const report = await this.scores.report(this.runId, day);
      this.cache.set(day, report);
      if (this.selectedDay() === day) {
        this.dayReport.set(report);
      }
    } catch {
      this.reportFailed.set(true);
    }
  }

  protected title(run: RunDetail): string {
    return run.endedBy === 'gave_up'
      ? `You called it on day ${run.days}`
      : `Bankrupt on day ${run.days}`;
  }
}
