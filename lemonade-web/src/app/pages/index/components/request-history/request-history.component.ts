import { DatePipe } from '@angular/common';
import { Component, inject, signal } from '@angular/core';

import { RequestLogEntry, RequestLogService } from '../../../../services/request-log.service';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { JsonViewComponent } from '../../../../shared/json-view/json-view.component';
import { MethodBadgeComponent } from '../../../../shared/method-badge/method-badge.component';
import { StatusPillComponent } from '../../../../shared/status-pill/status-pill.component';

/** Expandable list of every API call made by the page. */
@Component({
  selector: 'app-request-history',
  standalone: true,
  imports: [
    CardComponent,
    DatePipe,
    IconComponent,
    JsonViewComponent,
    MethodBadgeComponent,
    StatusPillComponent,
  ],
  templateUrl: './request-history.component.html',
  styleUrl: './request-history.component.scss',
})
export class RequestHistoryComponent {
  protected readonly log = inject(RequestLogService);

  // The newest entry is open by default; these track the user's overrides.
  private readonly opened = signal(new Set<number>());
  private readonly closed = signal(new Set<number>());

  isOpen(entry: RequestLogEntry): boolean {
    const isNewest = this.log.entries()[0]?.id === entry.id;
    return this.opened().has(entry.id) || (isNewest && !this.closed().has(entry.id));
  }

  toggle(entry: RequestLogEntry): void {
    const open = this.isOpen(entry);
    this.opened.update((ids) => withMember(ids, entry.id, !open));
    this.closed.update((ids) => withMember(ids, entry.id, open));
  }

  clear(): void {
    this.log.clear();
    this.opened.set(new Set());
    this.closed.set(new Set());
  }
}

function withMember(ids: Set<number>, id: number, present: boolean): Set<number> {
  const next = new Set(ids);
  if (present) {
    next.add(id);
  } else {
    next.delete(id);
  }
  return next;
}
