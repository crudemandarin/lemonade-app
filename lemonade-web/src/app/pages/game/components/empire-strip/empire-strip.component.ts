import { Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Goal } from '../../../../core/api.models';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** The era, and the next thing to work toward, with a link to the Empire page. */
@Component({
  selector: 'app-empire-strip',
  standalone: true,
  imports: [MoneyPipe, RouterLink],
  templateUrl: './empire-strip.component.html',
  styleUrl: './empire-strip.component.scss',
})
export class EmpireStripComponent {
  readonly era = input.required<number>();
  readonly eraName = input.required<string>();
  readonly nextGoal = input<Goal | null>(null);
}
