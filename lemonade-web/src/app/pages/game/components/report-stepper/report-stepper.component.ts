import { Component, computed, input, output } from '@angular/core';

import { DayReport, ReportSummary } from '../../../../core/api.models';
import { DayReportComponent } from '../day-report/day-report.component';

/**
 * Pages through a run's ended days: previous, next and jump-to-day above the day's
 * report. Used by the past-days drawer and the run detail page.
 */
@Component({
  selector: 'app-report-stepper',
  standalone: true,
  imports: [DayReportComponent],
  templateUrl: './report-stepper.component.html',
  styleUrl: './report-stepper.component.scss',
})
export class ReportStepperComponent {
  /** Every ended day, oldest first; null while the list loads. */
  readonly days = input.required<ReportSummary[] | null>();
  readonly selected = input<number | null>(null);
  readonly report = input<DayReport | null>(null);
  readonly failed = input(false);
  /** Shown when the run has no ended days. */
  readonly emptyText = input(
    "No days have ended yet. Each day you finish will show up here. Days played before this feature was added aren't kept.",
  );
  readonly selectDay = output<number>();

  private readonly position = computed(() => {
    const days = this.days() ?? [];
    return days.findIndex((d) => d.day === this.selected());
  });
  protected readonly previous = computed(() => this.days()?.[this.position() - 1]?.day ?? null);
  protected readonly next = computed(() => this.days()?.[this.position() + 1]?.day ?? null);

  protected onJump(event: Event): void {
    this.selectDay.emit(Number((event.target as HTMLSelectElement).value));
  }
}
