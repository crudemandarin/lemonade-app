import { Component, computed, input, signal } from '@angular/core';

import { MoneyPipe } from '../../../../shared/money.pipe';

const KEY = 'lemonade.victoryDismissed';

function dismissedRun(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return null;
  }
}

/**
 * "Global leader on day N": shown once a run has met the victory condition. The run keeps
 * playing and its score keeps growing, so this is a card the player can dismiss (per run),
 * not a screen that ends anything.
 */
@Component({
  selector: 'app-victory-banner',
  standalone: true,
  imports: [MoneyPipe],
  templateUrl: './victory-banner.component.html',
  styleUrl: './victory-banner.component.scss',
})
export class VictoryBannerComponent {
  readonly runId = input.required<string>();
  readonly wonOnDay = input.required<number>();
  readonly wonNetWorth = input.required<number>();

  private readonly dismissed = signal(dismissedRun());
  protected readonly visible = computed(
    () => this.wonOnDay() > 0 && this.dismissed() !== this.runId(),
  );

  protected dismiss(): void {
    try {
      localStorage.setItem(KEY, this.runId());
    } catch {
      // Storage can be blocked; the card just returns on reload.
    }
    this.dismissed.set(this.runId());
  }
}
