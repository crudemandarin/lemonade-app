import { ComponentFixture, TestBed } from '@angular/core/testing';

import { newGameView } from '../../../../core/testing/fixtures';
import { MarketPanelComponent, TradeRequest } from './market-panel.component';

describe('MarketPanelComponent', () => {
  let fixture: ComponentFixture<MarketPanelComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(MarketPanelComponent);
    const resources = newGameView().resources;
    resources[0] = { ...resources[0], stock: 4, previousPrice: 18 };
    fixture.componentRef.setInput('resources', resources);
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

  it('emits buy with the entered quantity', () => {
    const emitted: TradeRequest[] = [];
    fixture.componentInstance.buy.subscribe((t) => emitted.push(t));

    const input = row('ice').querySelector<HTMLInputElement>('.buy input')!;
    input.value = '3';
    row('ice').querySelector<HTMLButtonElement>('.buy button')!.click();

    expect(emitted).toEqual([{ resource: 'ice', qty: 3 }]);
  });

  it('emits sell with the entered quantity', () => {
    const emitted: TradeRequest[] = [];
    fixture.componentInstance.sell.subscribe((t) => emitted.push(t));

    row('lemonade').querySelector<HTMLButtonElement>('.sell button')!.click();

    expect(emitted).toEqual([{ resource: 'lemonade', qty: 1 }]);
  });
});
