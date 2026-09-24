import { Injectable, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Observable, Subject, catchError, map, of, switchMap, tap } from 'rxjs';

import { Sample } from '../models/sample.model';
import { apiErrorMessage } from '../utilities/api-error';
import { SampleService } from './sample.service';

type LoadResult = { samples: Sample[] } | { error: string };

/** Shared sample list state; every mutation reloads the list from the API. */
@Injectable({ providedIn: 'root' })
export class SampleStore {
  private readonly api = inject(SampleService);
  private readonly reload$ = new Subject<void>();

  private readonly _samples = signal<Sample[]>([]);
  private readonly _loaded = signal(false);
  private readonly _loading = signal(false);
  private readonly _error = signal<string | null>(null);

  readonly samples = this._samples.asReadonly();
  /** True once the list has been fetched successfully at least once. */
  readonly loaded = this._loaded.asReadonly();
  readonly loading = this._loading.asReadonly();
  readonly error = this._error.asReadonly();

  constructor() {
    this.reload$
      .pipe(
        tap(() => this._loading.set(true)),
        switchMap(() =>
          this.api.list().pipe(
            map((samples): LoadResult => ({ samples })),
            catchError((err) => of<LoadResult>({ error: apiErrorMessage(err) })),
          ),
        ),
        takeUntilDestroyed(),
      )
      .subscribe((result) => {
        this._loading.set(false);
        if ('error' in result) {
          this._error.set(result.error);
          return;
        }
        this._samples.set([...result.samples].sort((a, b) => a.id - b.id));
        this._loaded.set(true);
        this._error.set(null);
      });
  }

  load(): void {
    this.reload$.next();
  }

  create(name: string): Observable<Sample> {
    return this.api.create(name).pipe(tap(() => this.load()));
  }

  update(id: number, name: string): Observable<Sample> {
    return this.api.update(id, name).pipe(tap(() => this.load()));
  }

  delete(id: number): Observable<void> {
    return this.api.delete(id).pipe(tap(() => this.load()));
  }
}
