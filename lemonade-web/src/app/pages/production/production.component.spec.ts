import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Recipe } from '../../core/api.models';
import { newGameView } from '../../core/testing/fixtures';
import { ProductionComponent } from './production.component';

const recipe = (over: Partial<Recipe>): Recipe => ({
  key: 'lemonade',
  name: 'Lemonade',
  output: 'lemonade',
  outputQty: 1,
  inputs: [
    { resource: 'lemon', qty: 1 },
    { resource: 'sugar', qty: 1 },
  ],
  era: 0,
  learnCost: 0,
  text: 'The classic.',
  state: 'known',
  lockCode: '',
  lockedReason: '',
  ...over,
});

describe('ProductionComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<ProductionComponent>;
  let el: HTMLElement;

  const tick = () => new Promise((resolve) => setTimeout(resolve));

  async function render(capital = 5000) {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    fixture = TestBed.createComponent(ProductionComponent);
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
    http.expectOne('/api/game').flush({
      ...newGameView({ capital }),
      recipes: [
        recipe({}),
        recipe({ key: 'arnold_palmer', name: 'Arnold Palmer', era: 2, state: 'known' }),
        recipe({ key: 'limeade', name: 'Limeade', learnCost: 800, state: 'available' }),
        recipe({
          key: 'mint_lemonade',
          name: 'Mint lemonade',
          learnCost: 4000,
          state: 'locked',
          lockCode: 'era',
          lockedReason: 'Reach era 2 first',
        }),
      ],
      plan: [{ recipe: 'lemonade', target: 0 }],
    });
    await tick();
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  afterEach(() => http.verify());

  const text = (e: Element) => e.textContent!.replace(/\s+/g, ' ').trim();
  const item = (key: string) => el.querySelector<HTMLElement>(`[data-recipe=${key}]`)!;

  it('lists known, learnable and locked recipes, with the reason for a lock', async () => {
    await render();
    expect(text(item('lemonade'))).toContain('Known');
    expect(text(item('limeade'))).toContain('Learn for $800');
    expect(item('limeade').querySelector<HTMLButtonElement>('.learn')!.disabled).toBeFalse();
    expect(text(item('mint_lemonade'))).toContain('Reach era 2 first');
    expect(item('mint_lemonade').querySelector('.learn')).toBeNull();
  });

  it('cannot learn a recipe it cannot afford, and says how much is missing', async () => {
    await render(500);
    expect(item('limeade').querySelector<HTMLButtonElement>('.learn')!.disabled).toBeTrue();
    expect(text(item('limeade'))).toContain('Need $300 more');
  });

  it('asks before learning, then posts', async () => {
    await render();
    item('limeade').querySelector<HTMLButtonElement>('.learn')!.click();
    fixture.detectChanges();
    el.querySelector<HTMLButtonElement>(
      'app-confirm-dialog .btn-primary, app-confirm-dialog button:last-of-type',
    )!.click();
    const req = http.expectOne('/api/game/recipes/limeade/learn');
    expect(req.request.method).toBe('POST');
    req.flush(newGameView());
    await tick();
  });

  it('edits the plan locally and saves it in one request', async () => {
    await render();
    const save = () => el.querySelector<HTMLButtonElement>('.save')!;
    expect(save().disabled).toBeTrue();

    const select = el.querySelector<HTMLSelectElement>('select.add')!;
    expect(select.textContent).not.toContain('Mint lemonade');
    select.value = 'arnold_palmer';
    select.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    expect(el.querySelectorAll('.plan li').length).toBe(2);
    expect(save().disabled).toBeFalse();

    const target = el.querySelector<HTMLInputElement>('[data-plan=arnold_palmer] input')!;
    target.value = '30';
    target.dispatchEvent(new Event('change'));
    el.querySelector<HTMLButtonElement>('[data-plan=arnold_palmer] .up')!.click();
    fixture.detectChanges();
    expect(text(el.querySelector('.plan li')!)).toContain('1. Arnold Palmer');

    save().click();
    const req = http.expectOne('/api/game/production-plan');
    expect(req.request.body).toEqual({
      rows: [
        { recipe: 'arnold_palmer', target: 30 },
        { recipe: 'lemonade', target: 0 },
      ],
    });
    req.flush(newGameView());
    await tick();
  });

  it('discarding restores the saved plan', async () => {
    await render();
    const select = el.querySelector<HTMLSelectElement>('select.add')!;
    select.value = 'arnold_palmer';
    select.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    el.querySelector<HTMLButtonElement>('.reset')!.click();
    fixture.detectChanges();
    expect(el.querySelectorAll('.plan li').length).toBe(1);
  });
});
