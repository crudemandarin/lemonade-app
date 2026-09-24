import { Component, computed, input } from '@angular/core';

/** HTTP status code pill: green for 2xx/3xx, red for errors or no response. */
@Component({
  selector: 'app-status-pill',
  standalone: true,
  templateUrl: './status-pill.component.html',
  styleUrl: './status-pill.component.scss',
})
export class StatusPillComponent {
  readonly status = input.required<number>();

  protected readonly ok = computed(() => this.status() >= 200 && this.status() < 400);
}
