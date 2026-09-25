import { ComponentFixture, TestBed } from '@angular/core/testing';

import { newGameView } from '../../../../core/testing/fixtures';
import { ALL_QTY, MarketPanelComponent, TradeRequest } from './market-panel.component';

describe('MarketPanelComponent', () => {
  let fixture: ComponentFixture<MarketPanelComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    localStorage.removeItem('lemonade.tradeAmount');
    fixture = TestBed.createComponent(MarketPanelComponent);
    const resources = newGameView().resources;
    resources[0] = { ...resources[0], stock: 4, previousPrice: 18 };
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

  it('shows the ask on Buy and the bid on Sell', () => {
    expect(row('lemon').querySelector('.buy button')!.textContent).toContain('Buy $22');
    expect(row('lemon').querySelector('.sell button')!.textContent).toContain('Sell $18');
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

  it('offers 10, 50, 100 and All in a radio group, defaulting to 10', () => {
    expect(el.querySelector('[role=radiogroup]')).not.toBeNull();
    expect(radios().map((r) => r.value)).toEqual(['10', '50', '100', 'all']);
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

    const buttons = Array.from(el.querySelectorAll<HTMLButtonElement>('button'));
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
});
