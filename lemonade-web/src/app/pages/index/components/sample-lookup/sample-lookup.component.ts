import { DatePipe } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { finalize } from 'rxjs';

import { Sample } from '../../../../models/sample.model';
import { SampleService } from '../../../../services/sample.service';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { apiErrorMessage } from '../../../../utilities/api-error';

/** Single-sample lookup for GET /samples/:id. */
@Component({
  selector: 'app-sample-lookup',
  standalone: true,
  imports: [CardComponent, DatePipe, IconComponent],
  templateUrl: './sample-lookup.component.html',
  styleUrl: './sample-lookup.component.scss',
})
export class SampleLookupComponent {
  private readonly api = inject(SampleService);

  protected readonly id = signal('');
  protected readonly loading = signal(false);
  protected readonly result = signal<Sample | null>(null);
  protected readonly error = signal<string | null>(null);

  lookup(event: Event): void {
    event.preventDefault();
    this.result.set(null);
    this.error.set(null);

    const input = this.id().trim();
    const id = Number(input);
    if (!/^\d+$/.test(input) || id < 1) {
      this.error.set('ID must be a positive whole number.');
      return;
    }

    this.loading.set(true);
    this.api
      .get(id)
      .pipe(finalize(() => this.loading.set(false)))
      .subscribe({
        next: (sample) => this.result.set(sample),
        error: (err) => this.error.set(apiErrorMessage(err)),
      });
  }
}
