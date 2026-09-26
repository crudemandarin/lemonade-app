import { Component, OnInit, computed, inject, signal } from '@angular/core';

import { UpgradeItem, UpgradesResponse } from '../../core/api.models';
import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';
import { CardComponent } from '../../shared/card/card.component';
import { ConfirmDialogComponent } from '../../shared/confirm-dialog/confirm-dialog.component';
import { HelpLinkComponent } from '../../shared/help/help-link.component';
import { formatMoney, MoneyPipe } from '../../shared/money.pipe';

type Load = 'loading' | 'ready' | 'error';

/**
 * Every upgrade by category: what it costs, what it does, and whether it is owned,
 * available or locked (and why). Upgrades are permanent (they cannot be sold). The
 * page has no primary button: Buy is secondary, and asks first, showing the upkeep.
 */
@Component({
  selector: 'app-upgrades',
  standalone: true,
  imports: [CardComponent, ConfirmDialogComponent, HelpLinkComponent, MoneyPipe],
  templateUrl: './upgrades.component.html',
  styleUrl: './upgrades.component.scss',
})
export class UpgradesComponent implements OnInit {
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;

  protected readonly state = signal<Load>('loading');
  protected readonly data = signal<UpgradesResponse | null>(null);
  protected readonly confirming = signal<UpgradeItem | null>(null);

  protected readonly capital = computed(() => this.store.game()?.capital ?? 0);
  protected readonly active = computed(() => (this.store.game()?.status ?? 'active') === 'active');

  protected readonly groups = computed(() => {
    const data = this.data();
    if (!data) return [];
    return data.categories
      .map((c) => ({ ...c, items: data.upgrades.filter((u) => u.category === c.key) }))
      .filter((g) => g.items.length > 0);
  });

  ngOnInit(): void {
    void this.reload();
  }

  protected async reload(): Promise<void> {
    this.state.set('loading');
    try {
      if (!this.store.game()) {
        await this.store.load();
      }
      this.data.set(await this.store.upgradeList());
      this.state.set('ready');
    } catch {
      this.state.set('error');
    }
  }

  protected canBuy(u: UpgradeItem): boolean {
    return (
      u.state === 'available' &&
      this.active() &&
      this.online() &&
      !this.store.loading() &&
      this.capital() >= u.cost
    );
  }

  protected shortBy(u: UpgradeItem): number {
    return Math.max(0, u.cost - this.capital());
  }

  protected exact(value: number): string {
    return formatMoney(value);
  }

  protected question(u: UpgradeItem): string {
    const upkeep =
      u.upkeep > 0 ? `It costs ${formatMoney(u.upkeep)} a day to keep.` : 'It has no upkeep.';
    return `Buy ${u.name} for ${formatMoney(u.cost)}? ${upkeep} Upgrades cannot be sold.`;
  }

  protected async buy(u: UpgradeItem): Promise<void> {
    this.confirming.set(null);
    await this.store.buyUpgrade(u.key);
    if (this.store.error() === null) {
      await this.reload();
    }
  }
}
