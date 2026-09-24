import { Component, OnInit, inject, signal } from '@angular/core';

import { SampleService } from '../../../../services/sample.service';

type ApiState = 'checking' | 'online' | 'offline';

/** Health indicator backed by GET /; click to re-check. */
@Component({
  selector: 'app-api-status',
  standalone: true,
  templateUrl: './api-status.component.html',
  styleUrl: './api-status.component.scss',
})
export class ApiStatusComponent implements OnInit {
  private readonly api = inject(SampleService);

  protected readonly state = signal<ApiState>('checking');

  ngOnInit(): void {
    this.check();
  }

  check(): void {
    this.state.set('checking');
    this.api.health().subscribe({
      next: () => this.state.set('online'),
      error: () => this.state.set('offline'),
    });
  }
}
