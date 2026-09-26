import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { runSummary, scoreRow } from '../../core/testing/fixtures';
import { ScoresComponent } from './scores.component';

describe('ScoresComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<ScoresComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ScoresComponent);
    el = fixture.nativeElement;
    fixture.detectChanges();
  });

  afterEach(() => http.verify());

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };
  const tab = (name: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('[role=tab]')).find((t) =>
      t.textContent!.includes(name),
    )!;
  const text = () => el.textContent!.replace(/\s+/g, ' ');

  describe('global board', () => {
    it('shows a loading state, then the rows in a real table', async () => {
      expect(text()).toContain('Loading scores');
      http.expectOne('/api/scores').flush({
        rows: [
          scoreRow({ rank: 1, username: 'amy34', score: 2500, days: 14 }),
          scoreRow({ rank: 2, username: 'bob56', score: 1500, days: 6 }),
        ],
        me: null,
      });
      await settle();

      expect(el.querySelector('table')).not.toBeNull();
      const rows = Array.from(el.querySelectorAll('tbody tr')).map((r) =>
        r.textContent!.replace(/\s+/g, ' ').trim(),
      );
      expect(rows[0]).toContain('1');
      expect(rows[0]).toContain('amy34');
      expect(rows[0]).toContain('$2,500');
      expect(rows[0]).toContain('Day 14');
      expect(rows[1]).toContain('bob56');
      expect(el.querySelector('th[scope=col]')).not.toBeNull();
    });

    it('highlights the player, with text as well as tint', async () => {
      http.expectOne('/api/scores').flush({
        rows: [
          scoreRow({ rank: 1, username: 'amy34' }),
          scoreRow({ rank: 2, username: 'me123', isMe: true }),
        ],
        me: scoreRow({ rank: 2, username: 'me123', isMe: true }),
      });
      await settle();
      const mine = el.querySelectorAll('tbody tr.me');
      expect(mine.length).toBe(1);
      expect(mine[0].textContent).toContain('(you)');
      expect(mine[0].getAttribute('aria-current')).toBe('true');
    });

    it('shows the player below the top rows, after a gap', async () => {
      http.expectOne('/api/scores').flush({
        rows: [scoreRow({ rank: 1, username: 'amy34' })],
        me: scoreRow({ rank: 57, username: 'me123', score: 900, isMe: true }),
      });
      await settle();
      const rows = Array.from(el.querySelectorAll('tbody tr'));
      expect(rows.length).toBe(3);
      expect(rows[1].classList).toContain('gap');
      expect(rows[2].textContent).toContain('57');
      expect(rows[2].textContent).toContain('(you)');
    });

    it('says so when there are no finished runs', async () => {
      http.expectOne('/api/scores').flush({ rows: [], me: null });
      await settle();
      expect(el.querySelector('.empty')!.textContent).toContain('No finished runs yet');
      expect(el.querySelector('table')).toBeNull();
    });

    it('shows an error with a retry', async () => {
      http.expectOne('/api/scores').flush({}, { status: 500, statusText: '' });
      await settle();
      expect(el.querySelector('[role=alert]')!.textContent).toContain("Couldn't load the scores");
      el.querySelector<HTMLButtonElement>('.retry')!.click();
      http.expectOne('/api/scores').flush({ rows: [], me: null });
      await settle();
      expect(el.querySelector('[role=alert]')).toBeNull();
    });
  });

  describe('my runs', () => {
    beforeEach(async () => {
      http.expectOne('/api/scores').flush({ rows: [], me: null });
      await settle();
    });

    it('switches tabs accessibly', async () => {
      expect(tab('Global').getAttribute('aria-selected')).toBe('true');
      tab('Mine').click();
      http.expectOne('/api/runs').flush([]);
      await settle();
      expect(tab('Mine').getAttribute('aria-selected')).toBe('true');
      expect(tab('Global').getAttribute('aria-selected')).toBe('false');
      expect(el.querySelector('[role=tabpanel]')).not.toBeNull();
    });

    it('lists runs newest first, marks the best, and links each to its page', async () => {
      tab('Mine').click();
      http
        .expectOne('/api/runs')
        .flush([
          runSummary({ runId: 'r3', score: 1200, days: 4, endedBy: 'bankrupt' }),
          runSummary({ runId: 'r2', score: 2100, days: 9, endedBy: 'gave_up', isBest: true }),
        ]);
      await settle();

      const rows = el.querySelectorAll('tbody tr');
      expect(rows.length).toBe(2);
      expect(rows[0].textContent).toContain('$1,200');
      expect(rows[0].textContent).toContain('Went bankrupt');
      expect(rows[1].textContent).toContain('Gave up');
      expect(rows[1].querySelector('.best')!.textContent).toContain('Personal best');
      expect(rows[0].querySelector('.best')).toBeNull();
      expect(rows[1].querySelector('a')!.getAttribute('href')).toBe('/runs/r2');
    });

    it('says so when there are no runs yet', async () => {
      tab('Mine').click();
      http.expectOne('/api/runs').flush([]);
      await settle();
      expect(text()).toContain('No finished runs yet');
    });

    it('shows an error state', async () => {
      tab('Mine').click();
      http.expectOne('/api/runs').flush({}, { status: 500, statusText: '' });
      await settle();
      expect(el.querySelector('[role=alert]')!.textContent).toContain("Couldn't load your runs");
    });
  });
});

describe('ScoresComponent day 100 board', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<ScoresComponent>;
  let el: HTMLElement;

  beforeEach(async () => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ScoresComponent);
    el = fixture.nativeElement;
    fixture.detectChanges();
    http.expectOne('/api/scores').flush({
      board: 'all_time',
      rows: [
        scoreRow({ username: 'amy34', achievements: 7 }),
        scoreRow({ rank: 2, username: 'winner1', wonOnDay: 151 }),
      ],
      me: null,
    });
    await settle();
  });

  afterEach(() => http.verify());

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };
  const text = () => el.textContent!.replace(/\s+/g, ' ');

  it('marks a run that won the game, with the day in a tooltip', () => {
    const rows = el.querySelectorAll('tbody tr');
    expect(rows[0].querySelector('.won')).toBeNull();
    const badge = rows[1].querySelector('.won')!;
    expect(badge.textContent!.trim()).toBe('Won');
    expect(badge.getAttribute('title')).toContain('day 151');
  });

  it('shows each row with its achievement count as a number', () => {
    const row = el.querySelector('tbody tr')!;
    expect(row.querySelector('[data-label=Achievements]')!.textContent!.trim()).toBe('7');
    expect(text()).toContain('Achievements');
  });

  it('toggles to the day 100 board, marked by aria-pressed, and asks for it', async () => {
    const all = el.querySelector<HTMLButtonElement>('.board-all')!;
    const d100 = el.querySelector<HTMLButtonElement>('.board-100')!;
    expect(all.getAttribute('aria-pressed')).toBe('true');
    expect(d100.getAttribute('aria-pressed')).toBe('false');

    d100.click();
    http.expectOne('/api/scores?board=day_100').flush({
      board: 'day_100',
      rows: [scoreRow({ username: 'zed99', score: 48000, days: 100 })],
      me: null,
    });
    await settle();

    expect(d100.getAttribute('aria-pressed')).toBe('true');
    expect(all.getAttribute('aria-pressed')).toBe('false');
    expect(text()).toContain('Best net worth by day 100');
    expect(text()).toContain('zed99');
    expect(text()).toContain('$48,000');
    expect(text()).toContain('Net worth');
  });

  it('explains an empty day 100 board and why old runs are missing', async () => {
    el.querySelector<HTMLButtonElement>('.board-100')!.click();
    http.expectOne('/api/scores?board=day_100').flush({ board: 'day_100', rows: [], me: null });
    await settle();

    expect(el.querySelector('.empty')!.textContent).toContain('No run has reached day 100 yet');
    expect(el.querySelector('.empty')!.textContent).toContain('did not record it');
    expect(el.querySelector('table')).toBeNull();
  });

  it('goes back to all time without a board parameter', async () => {
    el.querySelector<HTMLButtonElement>('.board-100')!.click();
    http.expectOne('/api/scores?board=day_100').flush({ board: 'day_100', rows: [], me: null });
    await settle();
    el.querySelector<HTMLButtonElement>('.board-all')!.click();
    http.expectOne('/api/scores').flush({ board: 'all_time', rows: [], me: null });
    await settle();
    expect(text()).toContain('Global high scores');
  });
});
