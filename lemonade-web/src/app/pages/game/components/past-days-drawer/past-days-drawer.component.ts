import { Component, HostListener, computed, input, output } from '@angular/core';

import { DayReport, ReportSummary } from '../../../../core/api.models';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { DayReportComponent } from '../day-report/day-report.component';

/** A side drawer that pages back through the run's ended days, one report at a time. */
@Component({
  selector: 'app-past-days-drawer',
  standalone: true,
  imports: [DayReportComponent, IconComponent],
  templateUrl: './past-days-drawer.component.html',
  styleUrl: './past-days-drawer.component.scss',
})
export class PastDaysDrawerComponent {
  /** Every ended day, oldest first; null while the list loads. */
  readonly days = input.required<ReportSummary[] | null>();
  readonly selected = input<number | null>(null);
  readonly report = input<DayReport | null>(null);
  readonly failed = input(false);
  readonly selectDay = output<number>();
  readonly closed = output<void>();

  private readonly position = computed(() => {
    const days = this.days() ?? [];
    return days.findIndex((d) => d.day === this.selected());
  });
  protected readonly previous = computed(() => this.days()?.[this.position() - 1]?.day ?? null);
  protected readonly next = computed(() => this.days()?.[this.position() + 1]?.day ?? null);

  @HostListener('document:keydown.escape')
  protected onEscape(): void {
    this.closed.emit();
  }

  protected onJump(event: Event): void {
    this.selectDay.emit(Number((event.target as HTMLSelectElement).value));
  }
}
