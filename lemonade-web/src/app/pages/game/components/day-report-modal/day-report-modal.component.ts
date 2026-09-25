import { Component, input, output } from '@angular/core';

import { DayReport } from '../../../../core/api.models';
import { DayReportComponent } from '../day-report/day-report.component';

/** End-of-day summary in a modal, with the button that starts the next day. */
@Component({
  selector: 'app-day-report-modal',
  standalone: true,
  imports: [DayReportComponent],
  templateUrl: './day-report-modal.component.html',
  styleUrl: './day-report-modal.component.scss',
})
export class DayReportModalComponent {
  readonly report = input.required<DayReport>();
  readonly dismiss = output<void>();
}
