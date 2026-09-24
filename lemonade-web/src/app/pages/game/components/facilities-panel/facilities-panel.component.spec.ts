import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FacilityType, Resource } from '../../../../core/api.models';
import { newGameView } from '../../../../core/testing/fixtures';
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
      'Upgrade all warehouses to Garage: $500',
    );
    expect(warehouseCard().textContent).toContain('Covers 5 buildings at $100 each');
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

    const buttons = Array.from(el.querySelectorAll<HTMLButtonElement>('button'));
    expect(buttons.length).toBe(8);
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
});
