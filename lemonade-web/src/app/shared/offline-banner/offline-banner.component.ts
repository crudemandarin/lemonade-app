import { Component, inject } from '@angular/core';

import { OnlineService } from '../../core/online.service';
import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'app-offline-banner',
  standalone: true,
  imports: [IconComponent],
  template: `
    @if (!online()) {
      <p class="banner" role="status">
        <app-icon name="offline" />
        You are offline. Reconnect to keep playing.
      </p>
    }
  `,
  styles: `
    .banner {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      margin: 0;
      padding: 0.5rem 1rem;
      background: var(--text);
      color: var(--bg);
      font-size: 0.875rem;
    }
  `,
})
export class OfflineBannerComponent {
  protected readonly online = inject(OnlineService).online;
}
