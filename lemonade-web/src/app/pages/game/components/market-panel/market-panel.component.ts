import { Component, input, output, signal } from '@angular/core';

import { Resource, ResourceView } from '../../../../core/api.models';
import { RESOURCE_LABELS } from '../../../../core/resources';
import { CardComponent } from '../../../../shared/card/card.component';
import { HelpLinkComponent } from '../../../../shared/help/help-link.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { formatMoney, MoneyPipe } from '../../../../shared/money.pipe';
import { PriceSparklineComponent } from '../../../../shared/price-sparkline/price-sparkline.component';

export interface TradeRequest {
  resource: Resource;
  qty: number;
  /** Trade as many as possible up to qty instead of failing (see API `clamp`). */
  clamp: boolean;
}

/** Selectable trade amounts; `all` is sent as a huge quantity with clamp, so the server does the maths. */
export const TRADE_AMOUNTS = ['1', '10', '50', '100', 'all'] as const;
export type TradeAmount = (typeof TRADE_AMOUNTS)[number];
export const ALL_QTY = 1_000_000;
const STORAGE_KEY = 'lemonade.tradeAmount';
const DEFAULT_AMOUNT: TradeAmount = '10';

function loadAmount(): TradeAmount {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    return TRADE_AMOUNTS.find((a) => a === saved) ?? DEFAULT_AMOUNT;
  } catch {
    return DEFAULT_AMOUNT;
  }
}

/** Stock, prices, and buy/sell per resource, using one shared trade amount. Quantity checks happen on the server. */
@Component({
  selector: 'app-market-panel',
  standalone: true,
  imports: [HelpLinkComponent, CardComponent, IconComponent, MoneyPipe, PriceSparklineComponent],
  templateUrl: './market-panel.component.html',
  styleUrl: './market-panel.component.scss',
})
export class MarketPanelComponent {
  readonly resources = input.required<ResourceView[]>();
  readonly capital = input.required<number>();
  /** Disables every action, e.g. while offline. */
  readonly disabled = input(false);
  readonly buy = output<TradeRequest>();
  readonly sell = output<TradeRequest>();

  protected readonly labels = RESOURCE_LABELS;
  protected readonly amounts = TRADE_AMOUNTS;
  protected readonly amount = signal<TradeAmount>(loadAmount());

  protected select(amount: TradeAmount): void {
    this.amount.set(amount);
    try {
      localStorage.setItem(STORAGE_KEY, amount);
    } catch {
      // Storage can be blocked; the choice just won't persist.
    }
  }

  private request(row: ResourceView): TradeRequest {
    const amount = this.amount();
    return {
      resource: row.resource,
      qty: amount === 'all' ? ALL_QTY : Number(amount),
      clamp: true,
    };
  }

  protected onBuy(row: ResourceView): void {
    this.buy.emit(this.request(row));
  }

  protected onSell(row: ResourceView): void {
    this.sell.emit(this.request(row));
  }

  private wanted(): number {
    const amount = this.amount();
    return amount === 'all' ? ALL_QTY : Number(amount);
  }

  /**
   * Cases a Buy would trade, and what they cost: the selected amount, cut down to what
   * cash and warehouse space allow (the server applies the same limits with `clamp`).
   * Only for the button label; the server decides the real trade.
   */
  protected buyPreview(row: ResourceView): { qty: number; total: number } {
    const affordable = Math.floor(this.capital() / row.ask);
    const qty = Math.max(0, Math.min(this.wanted(), affordable, row.capacity - row.stock));
    return { qty, total: qty * row.ask };
  }

  protected sellPreview(row: ResourceView): { qty: number; total: number } {
    const qty = Math.min(this.wanted(), row.stock);
    return { qty, total: qty * row.bid };
  }

  protected canBuy(row: ResourceView): boolean {
    return !this.disabled() && row.stock < row.capacity && this.capital() >= row.ask;
  }

  protected canSell(row: ResourceView): boolean {
    return !this.disabled() && row.stock > 0;
  }

  /** Null on day 1, when there is no yesterday to compare against. */
  protected trend(row: ResourceView): 'up' | 'down' | 'flat' | null {
    if (row.previousPrice === null) {
      return null;
    }
    if (row.previousPrice === row.price) {
      return 'flat';
    }
    return row.price > row.previousPrice ? 'up' : 'down';
  }

  /** Signed text (not colour alone) for the gain against the current bid. */
  protected gainText(gain: number): string {
    if (gain === 0) {
      return 'even';
    }
    return `${gain > 0 ? '+' : '-'}${formatMoney(Math.abs(gain))}`;
  }

  protected gainLabel(gain: number): string {
    if (gain === 0) {
      return 'breaking even at the current bid';
    }
    return `${gain > 0 ? 'up' : 'down'} ${formatMoney(Math.abs(gain))} at the current bid`;
  }

  protected fill(row: ResourceView): number {
    return row.capacity > 0 ? Math.min(100, (row.stock / row.capacity) * 100) : 0;
  }
}
