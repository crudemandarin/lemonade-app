import { Component, HostListener, input, output } from '@angular/core';

import { DayReport, ReportSummary } from '../../../../core/api.models';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { ReportStepperComponent } from '../report-stepper/report-stepper.component';

/** A side drawer that pages back through the run's ended days, one report at a time. */
@Component({
  selector: 'app-past-days-drawer',
  standalone: true,
  imports: [IconComponent, ReportStepperComponent],
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

  @HostListener('document:keydown.escape')
  protected onEscape(): void {
    this.closed.emit();
  }
}
