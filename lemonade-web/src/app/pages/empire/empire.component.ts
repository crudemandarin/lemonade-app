import { Component, OnInit, computed, inject, signal } from '@angular/core';

import { CampaignOption, EmpireResponse, Rival, Territory } from '../../core/api.models';
import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';
import { RunNavComponent } from '../../shared/run-nav/run-nav.component';
import { CardComponent } from '../../shared/card/card.component';
import { ConfirmDialogComponent } from '../../shared/confirm-dialog/confirm-dialog.component';
import { HelpLinkComponent } from '../../shared/help/help-link.component';
import { formatMoney, MoneyPipe } from '../../shared/money.pipe';

type Load = 'loading' | 'ready' | 'error';

/** One thing the player is about to pay for; the dialog asks before anything is spent. */
interface Pending {
  title: string;
  message: string;
  confirmLabel: string;
  run: () => Promise<void>;
}

/**
 * The empire: every territory with the player's share and reach, the rivals in each, and
 * what can be done now (enter, run a campaign, buy a rival out). Every purchase asks first.
 */
@Component({
  selector: 'app-empire',
  standalone: true,
  imports: [RunNavComponent, CardComponent, ConfirmDialogComponent, HelpLinkComponent, MoneyPipe],
  templateUrl: './empire.component.html',
  styleUrl: './empire.component.scss',
})
export class EmpireComponent implements OnInit {
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;

  protected readonly state = signal<Load>('loading');
  protected readonly data = signal<EmpireResponse | null>(null);
  protected readonly pending = signal<Pending | null>(null);

  protected readonly capital = computed(() => this.store.game()?.capital ?? 0);
  protected readonly active = computed(() => (this.store.game()?.status ?? 'active') === 'active');
  protected readonly canAct = computed(
    () => this.active() && this.online() && !this.store.loading(),
  );

  ngOnInit(): void {
    void this.reload();
  }

  protected async reload(): Promise<void> {
    this.state.set('loading');
    try {
      if (!this.store.game()) {
        await this.store.load();
      }
      this.data.set(await this.store.empireDetail());
      this.state.set('ready');
    } catch {
      this.state.set('error');
    }
  }

  protected exact(value: number): string {
    return formatMoney(value);
  }

  protected activeRivals(t: Territory): Rival[] {
    return t.rivals.filter((r) => r.status === 'active');
  }

  protected doneRivals(t: Territory): Rival[] {
    return t.rivals.filter((r) => r.status !== 'active');
  }

  protected blockedReason(t: Territory): string {
    switch (t.enterBlocked) {
      case 'territory_locked':
        return 'Enter the territory before it first.';
      case 'insufficient_funds':
        return `Need ${formatMoney(t.entryCost - this.capital())} more.`;
      default:
        return '';
    }
  }

  protected askEnter(t: Territory): void {
    this.pending.set({
      title: `Enter ${t.name}?`,
      message: `Entering costs ${formatMoney(t.entryCost)} and starts you at ${t.entryShare}% of the market. It adds ${formatMoney(t.hubUpkeep)} a day in upkeep.`,
      confirmLabel: 'Enter',
      run: () => this.store.enterTerritory(t.key),
    });
  }

  protected askCampaign(t: Territory, c: CampaignOption): void {
    this.pending.set({
      title: `Run ${c.name}?`,
      message: `${c.name} in ${t.name} costs ${formatMoney(c.cost)} and lifts your share for ${c.days} days.`,
      confirmLabel: 'Run campaign',
      run: () => this.store.runCampaign(t.key, c.level),
    });
  }

  protected askBuyout(t: Territory, r: Rival, hostile: boolean): void {
    const price = hostile ? r.hostilePrice : r.buyoutPrice;
    this.pending.set({
      title: `${hostile ? 'Take over' : 'Buy out'} ${r.name}?`,
      message: `${hostile ? 'A hostile takeover of' : 'Buying'} ${r.name} in ${t.name} costs ${formatMoney(price)}. It cannot be undone.`,
      confirmLabel: hostile ? 'Take over' : 'Buy out',
      run: () => this.store.buyOutRival(r.key, hostile),
    });
  }

  protected async confirm(): Promise<void> {
    const p = this.pending();
    this.pending.set(null);
    if (!p) return;
    await p.run();
    if (this.store.error() === null) {
      await this.reload();
    }
  }
}
