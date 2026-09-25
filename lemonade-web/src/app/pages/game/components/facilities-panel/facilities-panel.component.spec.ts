import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FacilityType, Resource } from '../../../../core/api.models';
import { newGameView, noSale } from '../../../../core/testing/fixtures';
import { FacilitiesPanelComponent } from './facilities-panel.component';

describe('FacilitiesPanelComponent', () => {
  let fixture: ComponentFixture<FacilitiesPanelComponent>;
  let el: HTMLElement;

  function render(game = newGameView()) {
    fixture = TestBed.createComponent(FacilitiesPanelComponent);
    fixture.componentRef.setInput('warehouse', game.facilities.warehouse);
    fixture.componentRef.setInput('production', game.facilities.production);
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  const warehouseCard = () => el.querySelector<HTMLElement>('.warehouse')!;
  const productionCard = () => el.querySelector<HTMLElement>('.production')!;

  beforeEach(() => render());

  it('shows tier, level, and the upgrade cost for the whole type', () => {
    expect(warehouseCard().textContent).toContain('Pantry');
    expect(warehouseCard().textContent).toContain('level 1 of 4');
    expect(warehouseCard().querySelector('.upgrade')!.textContent).toContain(
      'Upgrade all warehouses to Garage: $775',
    );
    expect(warehouseCard().textContent).toContain('Covers 5 buildings at $155 each');
    expect(warehouseCard().textContent).toMatch(/market takes more[\s\S]*80 to\s*280/);
    expect(productionCard().textContent).toContain('Makes 10 lemonade per day');
  });

  it('emits expandWarehouse for the clicked resource', () => {
    const emitted: Resource[] = [];
    fixture.componentInstance.expandWarehouse.subscribe((r) => emitted.push(r));

    warehouseCard().querySelector<HTMLButtonElement>('[data-resource=sugar] button')!.click();

    expect(emitted).toEqual(['sugar']);
  });

  it('emits expandProduction', () => {
    let count = 0;
    fixture.componentInstance.expandProduction.subscribe(() => count++);

    productionCard().querySelector<HTMLButtonElement>('.expand')!.click();

    expect(count).toBe(1);
  });

  it('emits upgrade with the facility type', () => {
    const emitted: FacilityType[] = [];
    fixture.componentInstance.upgrade.subscribe((t) => emitted.push(t));

    warehouseCard().querySelector<HTMLButtonElement>('.upgrade')!.click();
    productionCard().querySelector<HTMLButtonElement>('.upgrade')!.click();

    expect(emitted).toEqual(['warehouse', 'production']);
  });

  it('shows Max level and does not emit at max level', () => {
    const game = newGameView();
    game.facilities.warehouse = { ...game.facilities.warehouse, level: 4, upgrade: null };
    render(game);
    const emitted: FacilityType[] = [];
    fixture.componentInstance.upgrade.subscribe((t) => emitted.push(t));

    const button = warehouseCard().querySelector<HTMLButtonElement>('.upgrade')!;
    button.click();

    expect(button.textContent).toContain('Max level');
    expect(emitted).toEqual([]);
  });

  it('disables every expand and upgrade button when disabled', () => {
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    const buttons = Array.from(el.querySelectorAll<HTMLButtonElement>('button:not(.help-link)'));
    expect(buttons.length).toBe(14);
    expect(buttons.every((b) => b.disabled)).toBeTrue();
  });

  it('shows the tier image for each facility type and level', () => {
    const game = newGameView();
    game.facilities.production = { ...game.facilities.production, level: 3 };
    render(game);

    const src = (card: HTMLElement) => card.querySelector('img.tier-img')!.getAttribute('src');
    expect(src(warehouseCard())).toBe('assets/facilities/warehouse-1.svg');
    expect(src(productionCard())).toBe('assets/facilities/production-3.svg');
  });

  it('shows each resource image in the warehouse rows', () => {
    const img = warehouseCard().querySelector('[data-resource=cup] img')!;
    expect(img.getAttribute('src')).toBe('assets/resources/cup.svg');
  });

  it('shows the upkeep increase on each upgrade button', () => {
    const text = (el: Element | null) => el!.textContent!.replace(/\s+/g, ' ');
    expect(text(fixture.nativeElement.querySelector('.warehouse .upgrade'))).toContain(
      '(increase $10 upkeep)',
    );
    expect(text(fixture.nativeElement.querySelector('.production .upgrade'))).toContain(
      '(increase $15 upkeep)',
    );
  });

  describe('selling buildings', () => {
    function withSellable() {
      const game = newGameView();
      const sellable = {
        sellValue: 50,
        canSell: true,
        sellBlockedReason: '' as const,
        casesToSell: 0,
      };
      game.facilities.warehouse.resources[1] = {
        ...game.facilities.warehouse.resources[1],
        count: 2,
        ...sellable,
      };
      game.facilities.production = {
        ...game.facilities.production,
        buildings: 2,
        ...sellable,
        sellValue: 250,
      };
      return game;
    }

    const sellButton = (card: HTMLElement, selector = '') =>
      card.querySelector<HTMLButtonElement>(`${selector} .sell-building`.trim())!;

    it('disables Sell for a last building, with the reason', () => {
      const button = sellButton(warehouseCard(), '[data-resource=lemon]');
      expect(button.disabled).toBeTrue();
      expect(button.title).toBe('Keep at least one');
    });

    it('explains when stock is in the way', () => {
      const game = newGameView();
      game.facilities.warehouse.resources[0] = {
        ...game.facilities.warehouse.resources[0],
        canSell: false,
        sellBlockedReason: 'stock_exceeds_capacity',
        casesToSell: 4,
      };
      render(game);
      expect(sellButton(warehouseCard(), '[data-resource=lemon]').title).toBe('Sell 4 cases first');
    });

    it('asks for confirmation, then emits the sale', () => {
      render(withSellable());
      const emitted: unknown[] = [];
      fixture.componentInstance.sellFacility.subscribe((s) => emitted.push(s));

      sellButton(warehouseCard(), '[data-resource=sugar]').click();
      fixture.detectChanges();
      expect(emitted).toEqual([]);
      expect(el.querySelector('[role=alertdialog]')!.textContent).toContain('Sell a Pantry?');

      el.querySelector<HTMLButtonElement>('[role=alertdialog] .danger')!.click();
      fixture.detectChanges();
      expect(emitted).toEqual([{ type: 'warehouse', resource: 'sugar' }]);
      expect(el.querySelector('[role=alertdialog]')).toBeNull();
    });

    it('cancel closes the dialog without selling', () => {
      render(withSellable());
      const emitted: unknown[] = [];
      fixture.componentInstance.sellFacility.subscribe((s) => emitted.push(s));

      sellButton(productionCard()).click();
      fixture.detectChanges();
      el.querySelector<HTMLButtonElement>('[role=alertdialog] .cancel')!.click();
      fixture.detectChanges();

      expect(emitted).toEqual([]);
      expect(el.querySelector('[role=alertdialog]')).toBeNull();
    });

    it('sells production without a resource', () => {
      render(withSellable());
      const emitted: unknown[] = [];
      fixture.componentInstance.sellFacility.subscribe((s) => emitted.push(s));
      sellButton(productionCard()).click();
      fixture.detectChanges();
      el.querySelector<HTMLButtonElement>('[role=alertdialog] .danger')!.click();
      expect(emitted).toEqual([{ type: 'production', resource: undefined }]);
    });

    it('a lone production building cannot be sold', () => {
      const game = newGameView();
      expect(game.facilities.production).toEqual(jasmine.objectContaining(noSale(250)));
      expect(sellButton(productionCard()).disabled).toBeTrue();
    });
  });
});
