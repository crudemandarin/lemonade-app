import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { dayReport, newGameView, timelinePoint } from '../../core/testing/fixtures';
import { GameComponent } from './game.component';

describe('GameComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<GameComponent>;

  async function render(overrides = {}) {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    fixture = TestBed.createComponent(GameComponent);
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
    http.expectOne('/api/game').flush(newGameView(overrides));
    await new Promise((resolve) => setTimeout(resolve)); // let the store finish loading
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  }

  afterEach(() => http.verify());

  it('shows the timeline charts on the game page while playing', async () => {
    const el = await render();
    expect(el.querySelector('app-timeline-charts')).not.toBeNull();
    expect(el.querySelector('app-game-over')).toBeNull();
  });

  it('shows the report, with the charts, once bankrupt', async () => {
    const el = await render({
      status: 'bankrupt',
      timeline: [timelinePoint(), timelinePoint({ kind: 'end_day' })],
    });
    expect(el.querySelector('app-game-over app-timeline-charts')).not.toBeNull();
    expect(el.querySelectorAll('app-timeline-charts').length).toBe(1);
  });

  describe('game timeline card', () => {
    beforeEach(() => localStorage.removeItem('lemonade.timelineOpen'));
    afterEach(() => localStorage.removeItem('lemonade.timelineOpen'));

    it('can be folded away and opened again', async () => {
      const el = await render();
      const toggle = () => el.querySelector<HTMLButtonElement>('.timeline-toggle')!;
      expect(toggle().getAttribute('aria-expanded')).toBe('true');
      expect(toggle().textContent).toContain('Hide');

      toggle().click();
      fixture.detectChanges();
      expect(el.querySelector('app-timeline-charts')).toBeNull();
      expect(toggle().getAttribute('aria-expanded')).toBe('false');
      expect(toggle().textContent).toContain('Show');
      expect(el.querySelector('app-card .card-header h2')).not.toBeNull(); // the title stays

      toggle().click();
      fixture.detectChanges();
      expect(el.querySelector('app-timeline-charts')).not.toBeNull();
    });

    it('remembers a folded timeline', async () => {
      localStorage.setItem('lemonade.timelineOpen', 'closed');
      const el = await render();
      expect(el.querySelector('app-timeline-charts')).toBeNull();
      expect(el.querySelector('.timeline-toggle')!.textContent).toContain('Show');
    });
  });

  it('shows the same result screen after giving up', async () => {
    const el = await render({ status: 'gave_up', day: 6 });
    expect(el.querySelector('app-game-over h1')!.textContent).toContain('You called it on day 6');
    expect(el.querySelector('.dashboard')).toBeNull();
  });

  describe('personal best', () => {
    it('shows the best score and its day while playing', async () => {
      const el = await render({ best: { score: 2100, days: 9, runId: 'old-run' } });
      expect(el.querySelector('.best-chip')!.textContent).toContain('Best: $2,100 on day 9');
    });

    it('shows no chip before a first finished run', async () => {
      const el = await render({ best: null });
      expect(el.querySelector('.best-chip')).toBeNull();
    });

    it('calls out a new personal best on the result screen', async () => {
      const el = await render({
        status: 'gave_up',
        runId: 'this-run',
        best: { score: 1500, days: 3, runId: 'this-run' },
      });
      expect(el.querySelector('.new-best')!.textContent).toContain('New personal best');
    });

    it('does not, when an earlier run is still the best', async () => {
      const el = await render({
        status: 'gave_up',
        runId: 'this-run',
        best: { score: 9000, days: 30, runId: 'old-run' },
      });
      expect(el.querySelector('.new-best')).toBeNull();
    });
  });

  describe('give up', () => {
    async function open() {
      const el = await render();
      el.querySelector<HTMLButtonElement>('.give-up button')!.click();
      fixture.detectChanges();
      return el;
    }

    it('is a quiet secondary button, never a second primary', async () => {
      const el = await render();
      expect(el.querySelectorAll('.btn-primary').length).toBe(1);
      expect(el.querySelector('.give-up .btn-primary')).toBeNull();
    });

    it('asks first, showing the score it would record, and does nothing on cancel', async () => {
      const el = await open();
      const dialog = el.querySelector('[role=alertdialog]')!;
      expect(dialog.textContent).toContain('Give up on day 1?');
      expect(dialog.textContent).toContain('$1,500');

      dialog.querySelector<HTMLButtonElement>('.cancel')!.click();
      fixture.detectChanges();
      expect(el.querySelector('[role=alertdialog]')).toBeNull();
      http.expectNone('/api/game/give-up');
    });

    it('posts to give-up once confirmed and shows the result screen', async () => {
      const el = await open();
      el.querySelector<HTMLButtonElement>('[role=alertdialog] .danger')!.click();
      http.expectOne('/api/game/give-up').flush(newGameView({ status: 'gave_up', day: 1 }));
      await new Promise((resolve) => setTimeout(resolve));
      fixture.detectChanges();

      expect(el.querySelector('app-game-over')).not.toBeNull();
      expect(el.querySelector('[role=alertdialog]')).toBeNull();
    });
  });

  describe('past days', () => {
    const summaries = [1, 2].map((day) => ({
      day,
      produced: 5,
      capitalBefore: 1000,
      capitalAfter: 970,
      newEvents: [],
      expiredEvents: [],
      bankrupt: false,
    }));
    const settle = async () => {
      await new Promise((resolve) => setTimeout(resolve));
      fixture.detectChanges();
    };

    it('opens on the latest day and pages back, fetching each day once', async () => {
      const el = await render();
      el.querySelector<HTMLButtonElement>('.past-days')!.click();
      fixture.detectChanges();
      http.expectOne('/api/game/reports').flush(summaries);
      await settle();
      http.expectOne('/api/game/reports/2').flush(dayReport({ day: 2 }));
      await settle();
      expect(el.querySelector('app-past-days-drawer h3')!.textContent).toContain('Day 2 report');

      el.querySelector<HTMLButtonElement>('app-past-days-drawer .prev')!.click();
      await settle();
      http.expectOne('/api/game/reports/1').flush(dayReport({ day: 1 }));
      await settle();
      expect(el.querySelector('app-past-days-drawer h3')!.textContent).toContain('Day 1 report');

      el.querySelector<HTMLButtonElement>('app-past-days-drawer .next')!.click();
      await settle();
      http.expectNone('/api/game/reports/2'); // cached
      expect(el.querySelector('app-past-days-drawer h3')!.textContent).toContain('Day 2 report');
    });

    it('shows the empty state for a run with no ended days', async () => {
      const el = await render();
      el.querySelector<HTMLButtonElement>('.past-days')!.click();
      fixture.detectChanges();
      http.expectOne('/api/game/reports').flush([]);
      await settle();
      expect(el.querySelector('app-past-days-drawer .empty')).not.toBeNull();
    });

    it('says so when the list cannot be loaded', async () => {
      const el = await render();
      el.querySelector<HTMLButtonElement>('.past-days')!.click();
      fixture.detectChanges();
      http.expectOne('/api/game/reports').flush({}, { status: 500, statusText: '' });
      await settle();
      expect(el.querySelector('app-past-days-drawer [role=alert]')).not.toBeNull();
    });
  });

  it("offers to repeat yesterday's trades only with the order book, and repeats them", async () => {
    const trades = [
      timelinePoint({ kind: 'buy', resource: 'lemon', qty: 4, day: 2 }),
      timelinePoint({ kind: 'sell', resource: 'lemonade', qty: 3, day: 2 }),
      timelinePoint({ kind: 'buy', resource: 'sugar', qty: 9, day: 1 }),
    ];
    let el = await render({ day: 3, timeline: trades });
    expect(el.querySelector('.repeat-trades')).toBeNull();
    http.verify();
    fixture.destroy();
    TestBed.resetTestingModule();

    el = await render({ day: 3, timeline: trades, features: ['repeat_trades'] });
    el.querySelector<HTMLButtonElement>('.repeat-trades')!.click();
    const buy = http.expectOne('/api/game/buy');
    expect(buy.request.body).toEqual({ resource: 'lemon', qty: 4, clamp: true });
    buy.flush(newGameView({ day: 3, features: ['repeat_trades'] }));
    await new Promise((resolve) => setTimeout(resolve));
    const sell = http.expectOne('/api/game/sell');
    expect(sell.request.body).toEqual({ resource: 'lemonade', qty: 3, clamp: true });
    sell.flush(newGameView({ day: 3, features: ['repeat_trades'] }));
    await new Promise((resolve) => setTimeout(resolve));
  });

  it('shows the forecast in the events banner', async () => {
    const el = await render({
      forecast: [{ daysAhead: 1, key: 'heat_wave', name: 'Heat Wave', duration: 2 }],
    });
    expect(el.querySelector('.forecast')!.textContent).toContain('Tomorrow: Heat Wave');
  });
});
