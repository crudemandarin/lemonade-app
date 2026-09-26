import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { newGameView, upgradesResponse } from '../../core/testing/fixtures';
import { UpgradesComponent } from './upgrades.component';

describe('UpgradesComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<UpgradesComponent>;
  let el: HTMLElement;

  const tick = () => new Promise((resolve) => setTimeout(resolve));

  async function render(capital = 1000) {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    fixture = TestBed.createComponent(UpgradesComponent);
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
    http.expectOne('/api/game').flush(newGameView({ capital }));
    await tick();
    http.expectOne('/api/game/upgrades').flush(upgradesResponse());
    await tick();
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  afterEach(() => http.verify());

  const item = (key: string) => el.querySelector<HTMLElement>(`[data-upgrade=${key}]`)!;
  const text = (e: Element) => e.textContent!.replace(/\s+/g, ' ').trim();

  it('lists upgrades by category with cost, upkeep and what they do', async () => {
    await render(5000);
    expect(el.querySelectorAll('app-card').length).toBe(2);
    expect(text(item('freezer_1'))).toContain('Chest freezer');
    expect(text(item('freezer_1'))).toContain('Cost $1,200');
    expect(text(item('freezer_1'))).toContain('Upkeep $5 a day');
    expect(text(item('freezer_1'))).toContain('Keep up to 20 cases of ice one extra night.');
    expect(text(item('order_book'))).toContain('No upkeep');
  });

  it('says in words why an upgrade is locked, and offers no button', async () => {
    await render(50_000);
    expect(text(item('freezer_2'))).toContain('Locked');
    expect(text(item('freezer_2'))).toContain('Reach era 2 first.');
    expect(item('freezer_2').querySelector('button')).toBeNull();
  });

  it('disables Buy and says how much more is needed when cash is short', async () => {
    await render(1000);
    const buy = item('freezer_1').querySelector<HTMLButtonElement>('button.buy')!;
    expect(buy.disabled).toBeTrue();
    expect(text(item('freezer_1'))).toContain('Need $200 more');
    expect(item('order_book').querySelector<HTMLButtonElement>('button.buy')!.disabled).toBeFalse();
  });

  it('has no primary button', async () => {
    await render(5000);
    expect(el.querySelector('.btn-primary')).toBeNull();
  });

  it('asks first, shows the upkeep, then buys and reloads the list', async () => {
    await render(5000);
    item('freezer_1').querySelector<HTMLButtonElement>('button.buy')!.click();
    fixture.detectChanges();
    const dialog = el.querySelector('app-confirm-dialog')!;
    expect(text(dialog)).toContain('Buy Chest freezer for $1,200?');
    expect(text(dialog)).toContain('It costs $5 a day to keep.');
    http.expectNone('/api/game/upgrades/freezer_1/buy');

    dialog.querySelector<HTMLButtonElement>('.btn-primary, button:last-of-type')!.click();
    const req = http.expectOne('/api/game/upgrades/freezer_1/buy');
    req.flush(newGameView({ capital: 3800 }));
    await tick();
    const list = http.expectOne('/api/game/upgrades');
    list.flush(
      upgradesResponse({
        ownedCount: 1,
        upgrades: upgradesResponse().upgrades.map((u) =>
          u.key === 'freezer_1' ? { ...u, state: 'owned' as const } : u,
        ),
      }),
    );
    await tick();
    fixture.detectChanges();
    expect(text(item('freezer_1'))).toContain('Owned');
  });
});
