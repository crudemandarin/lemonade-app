import { Injectable, signal } from '@angular/core';

const MAX_ENTRIES = 25;

export interface RequestLogEntry {
  id: number;
  method: string;
  url: string;
  /** HTTP status, or 0 when the request never got a response. */
  status: number;
  durationMs: number;
  timestamp: Date;
  requestBody: unknown;
  responseBody: unknown;
}

/** In-memory history of API calls, newest first. */
@Injectable({ providedIn: 'root' })
export class RequestLogService {
  private nextId = 1;

  readonly entries = signal<RequestLogEntry[]>([]);

  add(entry: Omit<RequestLogEntry, 'id'>): void {
    this.entries.update((entries) =>
      [{ ...entry, id: this.nextId++ }, ...entries].slice(0, MAX_ENTRIES),
    );
  }

  clear(): void {
    this.entries.set([]);
  }
}
