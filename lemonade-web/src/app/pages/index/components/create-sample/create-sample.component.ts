import { Component, inject, signal } from '@angular/core';
import { finalize } from 'rxjs';

import { SampleStore } from '../../../../services/sample.store';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { apiErrorMessage } from '../../../../utilities/api-error';

/** Form for POST /samples. */
@Component({
  selector: 'app-create-sample',
  standalone: true,
  imports: [CardComponent, IconComponent],
  templateUrl: './create-sample.component.html',
})
export class CreateSampleComponent {
  private readonly store = inject(SampleStore);

  protected readonly name = signal('');
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);

  submit(event: Event): void {
    event.preventDefault();
    this.saving.set(true);
    this.error.set(null);
    // Empty names are sent as-is so the API's validation error is shown.
    this.store
      .create(this.name().trim())
      .pipe(finalize(() => this.saving.set(false)))
      .subscribe({
        next: () => this.name.set(''),
        error: (err) => this.error.set(apiErrorMessage(err)),
      });
  }
}
