import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ResourceView, TradeQuote } from '../../../../core/api.models';
import { newGameView, tradeLadder } from '../../../../core/testing/fixtures';
import { ALL_QTY, MarketPanelComponent, TradeRequest } from './market-panel.component';

/** A row holding `stock` cases, with the trade ladder the server would send for it. */
function held(
  row: ResourceView,
  stock: number,
  overrides: Partial<ResourceView> = {},
): ResourceView {
  return {
    ...row,
    stock,
    trade: tradeLadder(row.ask, row.bid, stock, row.capacity),
    ...overrides,
  };
}

describe('MarketPanelComponent', () => {
  let fixture: ComponentFixture<MarketPanelComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    localStorage.removeItem('lemonade.tradeAmount');
    fixture = TestBed.createComponent(MarketPanelComponent);
    const resources = newGameView().resources;
    resources[0] = held(resources[0], 4, { previousPrice: 18 });
    fixture.componentRef.setInput('resources', resources);
    fixture.componentRef.setInput('capital', 1000);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  const row = (resource: string) => el.querySelector<HTMLElement>(`[data-resource=${resource}]`)!;

  it('renders one row per resource with stock over capacity', () => {
    expect(el.querySelectorAll('[data-resource]').length).toBe(5);
    expect(row('lemon').textContent).toContain('4 / 10');
    expect(row('cup').textContent).toContain('Cups');
  });

  const label = (resource: string, side: 'buy' | 'sell') =>
    row(resource).querySelector(`.${side} button`)!.textContent!.replace(/\s+/g, ' ').trim();

  it('prices the buy and sell buttons for the selected amount', () => {
    // Sugar: ask $11, bid $9, empty warehouse with room for 10, $1,000 in cash.
    expect(label('sugar', 'buy')).toBe('Buy 10 · $110');
    pick('50');
    // Only 10 fit, so that is what the button offers and prices.
    expect(label('sugar', 'buy')).toBe('Buy 10 · $110');
    // Lemon holds 4 of 10 (ask $22, bid $18): 6 fit, 4 can be sold.
    expect(label('lemon', 'buy')).toBe('Buy 6 · $132');
    expect(label('lemon', 'sell')).toBe('Sell 4 · $72');
  });

  it('shows the unit price when nothing can be traded', () => {
    expect(label('sugar', 'sell')).toBe('Sell $9'); // no stock
    const resources = newGameView().resources;
    resources[1] = { ...resources[1], trade: tradeLadder(11, 9, 0, 10, 5) }; // $5 buys nothing
    fixture.componentRef.setInput('resources', resources);
    fixture.detectChanges();
    expect(label('sugar', 'buy')).toBe('Buy $11');
  });

  it('shows what the server says the buy costs, cut down to cash', () => {
    const resources = newGameView().resources;
    resources[1] = { ...resources[1], trade: tradeLadder(11, 9, 0, 10, 40) }; // $40 buys 3
    fixture.componentRef.setInput('resources', resources);
    fixture.detectChanges();
    expect(label('sugar', 'buy')).toBe('Buy 3 · $33');
  });

  it('follows the amount, including All', () => {
    const resources = newGameView().resources;
    resources[0] = held(resources[0], 10);
    fixture.componentRef.setInput('resources', resources);
    fixture.detectChanges();
    pick('all');
    expect(label('lemon', 'sell')).toBe('Sell 10 · $180');
    expect(label('sugar', 'buy')).toBe('Buy 10 · $110');
  });

  describe('price impact', () => {
    const slipped = (qty: number, total: number, avg: number, pct: number): TradeQuote => ({
      qty,
      total,
      averagePrice: avg,
      slippagePercent: pct,
    });

    function withImpact() {
      const resources = newGameView().resources;
      const base = resources[1];
      resources[1] = {
        ...base,
        ask: 12,
        buyImpactPercent: 9,
        sellImpactPercent: 4,
        buyDepthLeft: 0,
        trade: {
          ...base.trade,
          buy: { ...base.trade.buy, '10': slipped(10, 128, 12.8, 16.4) },
        },
      };
      fixture.componentRef.setInput('resources', resources);
      fixture.detectChanges();
    }

    it('shows the price the server quoted, not unit price times amount', () => {
      withImpact();
      expect(label('sugar', 'buy')).toBe('Buy 10 · $128');
    });

    it('explains the slippage in words on the button', () => {
      withImpact();
      const button = row('sugar').querySelector<HTMLButtonElement>('.buy button')!;
      expect(button.title).toBe('Average $12.8 a case, 16.4% above the plain price');
      expect(row('lemon').querySelector<HTMLButtonElement>('.buy button')!.title).toBe('');
    });

    it('marks a row whose price the player has moved, with the words as well as an icon', () => {
      withImpact();
      const hint = row('sugar').querySelector('.impact')!;
      expect(hint.getAttribute('aria-label')).toContain('ask +9%');
      expect(hint.getAttribute('aria-label')).toContain('bid -4%');
      expect(hint.getAttribute('title')).toContain('wears off overnight');
      expect(row('sugar').querySelector('.impact app-icon')).not.toBeNull();
      expect(row('lemon').querySelector('.impact')).toBeNull();
    });
  });

  it('shows a trend arrow against yesterday', () => {
    expect(row('lemon').querySelector('.trend.up')).not.toBeNull();
    expect(row('sugar').querySelector('.trend.up, .trend.down')).toBeNull();
  });

  const radios = () => Array.from(el.querySelectorAll<HTMLInputElement>('input[type=radio]'));
  const pick = (value: string) => {
    radios()
      .find((r) => r.value === value)!
      .click();
    fixture.detectChanges();
  };

  it('offers 1, 10, 50, 100 and All in a radio group, defaulting to 10', () => {
    expect(el.querySelector('[role=radiogroup]')).not.toBeNull();
    expect(radios().map((r) => r.value)).toEqual(['1', '10', '50', '100', 'all']);
    expect(radios().find((r) => r.checked)!.value).toBe('10');
  });

  it('buys and sells the selected amount with clamp', () => {
    const bought: TradeRequest[] = [];
    const sold: TradeRequest[] = [];
    fixture.componentInstance.buy.subscribe((t) => bought.push(t));
    fixture.componentInstance.sell.subscribe((t) => sold.push(t));

    row('lemon').querySelector<HTMLButtonElement>('.buy button')!.click();
    pick('50');
    row('lemon').querySelector<HTMLButtonElement>('.sell button')!.click();
    pick('all');
    row('lemon').querySelector<HTMLButtonElement>('.buy button')!.click();

    expect(bought).toEqual([
      { resource: 'lemon', qty: 10, clamp: true },
      { resource: 'lemon', qty: ALL_QTY, clamp: true },
    ]);
    expect(sold).toEqual([{ resource: 'lemon', qty: 50, clamp: true }]);
  });

  it('remembers the selection', () => {
    pick('100');
    expect(localStorage.getItem('lemonade.tradeAmount')).toBe('100');

    const again = TestBed.createComponent(MarketPanelComponent);
    again.componentRef.setInput('resources', newGameView().resources);
    again.componentRef.setInput('capital', 1000);
    again.detectChanges();
    const checked = again.nativeElement.querySelector(
      'input[type=radio]:checked',
    ) as HTMLInputElement;
    expect(checked.value).toBe('100');
  });

  it('ignores a junk stored value', () => {
    localStorage.setItem('lemonade.tradeAmount', 'lots');
    const again = TestBed.createComponent(MarketPanelComponent);
    again.componentRef.setInput('resources', newGameView().resources);
    again.componentRef.setInput('capital', 1000);
    again.detectChanges();
    const checked = again.nativeElement.querySelector(
      'input[type=radio]:checked',
    ) as HTMLInputElement;
    expect(checked.value).toBe('10');
  });

  it('disables a row button when the action is impossible', () => {
    // lemon: stock 4 (can sell, can buy); the others have no stock to sell.
    expect(row('lemon').querySelector<HTMLButtonElement>('.sell button')!.disabled).toBeFalse();
    expect(row('sugar').querySelector<HTMLButtonElement>('.sell button')!.disabled).toBeTrue();

    fixture.componentRef.setInput('capital', 5);
    fixture.detectChanges();
    expect(row('lemon').querySelector<HTMLButtonElement>('.buy button')!.disabled).toBeTrue();

    const full = newGameView().resources;
    full[1] = { ...full[1], stock: full[1].capacity };
    fixture.componentRef.setInput('capital', 1000);
    fixture.componentRef.setInput('resources', full);
    fixture.detectChanges();
    expect(row('sugar').querySelector<HTMLButtonElement>('.buy button')!.disabled).toBeTrue();
  });

  it('disables every buy and sell button when disabled', () => {
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    const buttons = Array.from(el.querySelectorAll<HTMLButtonElement>('button:not(.help-link)'));
    expect(buttons.length).toBe(10);
    expect(buttons.every((b) => b.disabled)).toBeTrue();
  });

  it('shows each resource image', () => {
    const img = row('ice').querySelector<HTMLImageElement>('img.resource-img')!;
    expect(img.getAttribute('src')).toBe('assets/resources/ice.svg');
  });

  it('shows a flat trend when the price did not move', () => {
    const resources = newGameView().resources;
    resources[1] = { ...resources[1], previousPrice: resources[1].price };
    fixture.componentRef.setInput('resources', resources);
    fixture.detectChanges();
    expect(row('sugar').querySelector('.trend.flat')).not.toBeNull();
  });

  it('draws a price-history sparkline on every row', () => {
    expect(el.querySelectorAll('app-price-sparkline').length).toBe(5);
    expect(
      row('lemon').querySelector('app-price-sparkline svg')!.getAttribute('aria-label'),
    ).toContain('Lemon');
  });

  it('keeps the stock widget compact: numbers plus a slim bar', () => {
    const stock = row('lemon').querySelector('.stock')!;
    expect(stock.textContent).toContain('4 / 10');
    expect(stock.querySelector('[role=progressbar]')).not.toBeNull();
  });

  describe('average cost', () => {
    function withStock(overrides: Partial<ResourceView>) {
      const resources = newGameView().resources;
      resources[0] = { ...resources[0], stock: 4, ...overrides };
      fixture.componentRef.setInput('resources', resources);
      fixture.detectChanges();
      return row('lemon').querySelector('.cost-text')!.textContent!.replace(/\s+/g, ' ').trim();
    }

    it('shows the average and a signed gain for a profit', () => {
      expect(withStock({ avgCost: 18, unrealizedGain: 12 })).toBe('avg $18 +$12');
      expect(row('lemon').querySelector('.gain.up')).not.toBeNull();
    });

    it('shows a signed loss', () => {
      expect(withStock({ avgCost: 22, unrealizedGain: -16 })).toBe('avg $22 -$16');
      expect(row('lemon').querySelector('.gain.down')).not.toBeNull();
    });

    it('says even at zero, and names the cost to make on lemonade', () => {
      expect(withStock({ avgCost: 20, unrealizedGain: 0 })).toBe('avg $20 even');
      const resources = newGameView().resources;
      resources[4] = { ...resources[4], stock: 2, avgCost: 55, unrealizedGain: 30 };
      fixture.componentRef.setInput('resources', resources);
      fixture.detectChanges();
      expect(row('lemonade').querySelector('.cost-text')!.textContent).toContain(
        'cost to make $55',
      );
    });

    it('shows no figures when no stock is held', () => {
      expect(row('sugar').querySelector('.cost-text')!.textContent!.trim()).toBe('');
      expect(row('sugar').querySelector('.gain')).toBeNull();
    });

    it('keeps the stock cell the same height with or without stock', () => {
      // Lemon holds 4, sugar holds 0: buying into an empty row must not resize it.
      const held = row('lemon').querySelector<HTMLElement>('.stock')!.offsetHeight;
      const empty = row('sugar').querySelector<HTMLElement>('.stock')!.offsetHeight;
      expect(empty).toBe(held);
      const rows = newGameView().resources;
      rows[1] = { ...rows[1], stock: 3, avgCost: 10, unrealizedGain: -3 };
      fixture.componentRef.setInput('resources', rows);
      fixture.detectChanges();
      expect(row('sugar').querySelector<HTMLElement>('.stock')!.offsetHeight).toBe(empty);
    });
  });

  describe('upgrades', () => {
    it('shows the 7-day average only when the game sends one', () => {
      expect(row('sugar').querySelector('.avg')).toBeNull();
      const resources = newGameView().resources;
      resources[1] = { ...resources[1], movingAverage: 12 };
      fixture.componentRef.setInput('resources', resources);
      fixture.detectChanges();
      expect(row('sugar').querySelector('.avg')!.textContent).toContain('7-day avg $12');
    });

    it('has no alert inputs without the upgrade', () => {
      expect(el.querySelector('.alert-set')).toBeNull();
    });

    it('marks an input row in words when its ask is at or under the alert level', () => {
      fixture.componentRef.setInput('alertsOn', true);
      fixture.componentRef.setInput('alerts', { sugar: 11, lemon: 5, lemonade: 200 });
      fixture.detectChanges();
      expect(row('sugar').querySelector('.alert-hit')!.textContent).toContain(
        'Alert: ask is $11, at or under $11',
      );
      expect(row('lemon').querySelector('.alert-hit')).toBeNull();
      expect(row('lemonade').querySelector('.alert-hit')).toBeNull();
      fixture.componentRef.setInput('alerts', { lemonade: 80 });
      fixture.detectChanges();
      expect(row('lemonade').querySelector('.alert-hit')!.textContent).toContain('bid is $90');
    });

    it('emits the level typed, and null when it is cleared', () => {
      fixture.componentRef.setInput('alertsOn', true);
      fixture.detectChanges();
      const seen: { resource: string; level: number | null }[] = [];
      fixture.componentInstance.alertChange.subscribe((c) => seen.push(c));
      const input = row('sugar').querySelector<HTMLInputElement>('.alert-set input')!;
      input.value = '9';
      input.dispatchEvent(new Event('change'));
      input.value = '';
      input.dispatchEvent(new Event('change'));
      expect(seen).toEqual([
        { resource: 'sugar', level: 9 },
        { resource: 'sugar', level: null },
      ]);
    });
  });
});
