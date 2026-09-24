import { animate, style, transition, trigger } from '@angular/animations';
import { DatePipe } from '@angular/common';
import { Component, OnInit, effect, inject, signal } from '@angular/core';
import { finalize } from 'rxjs';

import { Sample } from '../../../../models/sample.model';
import { SampleStore } from '../../../../services/sample.store';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { apiErrorMessage } from '../../../../utilities/api-error';

const HIGHLIGHT_MS = 1600;

interface RowError {
  id: number;
  message: string;
}

/** Sample list (GET /samples) with inline edit (PUT) and delete (DELETE). */
@Component({
  selector: 'app-sample-table',
  standalone: true,
  imports: [CardComponent, DatePipe, IconComponent],
  templateUrl: './sample-table.component.html',
  styleUrl: './sample-table.component.scss',
  animations: [
    trigger('rowLeave', [
      transition(':leave', [
        animate('250ms ease-in', style({ opacity: 0, transform: 'translateX(1rem)' })),
      ]),
    ]),
  ],
})
export class SampleTableComponent implements OnInit {
  protected readonly store = inject(SampleStore);

  protected readonly editingId = signal<number | null>(null);
  protected readonly draft = signal('');
  protected readonly confirmingId = signal<number | null>(null);
  protected readonly busyId = signal<number | null>(null);
  protected readonly rowError = signal<RowError | null>(null);
  /** Rows that are new or changed since the previous load, briefly highlighted. */
  protected readonly highlightedIds = signal<ReadonlySet<number>>(new Set());

  private knownVersions = new Map<number, string>();
  private hasBaseline = false;
  private highlightTimer?: ReturnType<typeof setTimeout>;

  constructor() {
    effect(() => this.highlightChanges(this.store.samples(), this.store.loaded()), {
      allowSignalWrites: true,
    });
  }

  ngOnInit(): void {
    this.store.load();
  }

  startEdit(sample: Sample): void {
    this.editingId.set(sample.id);
    this.draft.set(sample.name);
    this.confirmingId.set(null);
    this.rowError.set(null);
  }

  cancelEdit(): void {
    this.editingId.set(null);
    this.rowError.set(null);
  }

  save(sample: Sample): void {
    this.busyId.set(sample.id);
    this.rowError.set(null);
    this.store
      .update(sample.id, this.draft().trim())
      .pipe(finalize(() => this.busyId.set(null)))
      .subscribe({
        next: () => this.editingId.set(null),
        error: (err) => this.rowError.set({ id: sample.id, message: apiErrorMessage(err) }),
      });
  }

  askDelete(sample: Sample): void {
    this.confirmingId.set(sample.id);
    this.editingId.set(null);
    this.rowError.set(null);
  }

  delete(sample: Sample): void {
    this.busyId.set(sample.id);
    this.store
      .delete(sample.id)
      .pipe(
        finalize(() => {
          this.busyId.set(null);
          this.confirmingId.set(null);
        }),
      )
      .subscribe({
        error: (err) => this.rowError.set({ id: sample.id, message: apiErrorMessage(err) }),
      });
  }

  /** Compares each sample's updated_at with the last list seen to find new or edited rows. */
  private highlightChanges(samples: Sample[], loaded: boolean): void {
    const changed = this.hasBaseline
      ? samples.filter((s) => this.knownVersions.get(s.id) !== s.updated_at).map((s) => s.id)
      : [];
    this.knownVersions = new Map(samples.map((s) => [s.id, s.updated_at]));
    this.hasBaseline = loaded;

    if (changed.length) {
      clearTimeout(this.highlightTimer);
      this.highlightedIds.set(new Set(changed));
      this.highlightTimer = setTimeout(() => this.highlightedIds.set(new Set()), HIGHLIGHT_MS);
    }
  }
}
