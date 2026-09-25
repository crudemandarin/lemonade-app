import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { GameStore } from './game.store';
import { FakeAuthPort, provideFakeAuth } from './testing/fake-auth';
import { SessionService } from './session.service';
import { ToastService } from './toast.service';
import { dayReport, newGameView } from './testing/fixtures';

describe('GameStore', () => {
  let store: GameStore;
  let http: HttpTestingController;
  let session: SessionService;
  let port: FakeAuthPort;

  beforeEach(() => {
    port = new FakeAuthPort();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideFakeAuth(port)],
    });
    store = TestBed.inject(GameStore);
    http = TestBed.inject(HttpTestingController);
    session = TestBed.inject(SessionService);
    session.signOut();
  });

  afterEach(() => {
    http.verify();
    session.signOut();
  });

  it('signIn plays as a guest on a username alone and resolves true', async () => {
    const done = store.signIn('lemonjoe');
    http.expectOne('/api/login').flush({ id: 1, username: 'lemonjoe' });

    expect(await done).toBeTrue();
    expect(session.username()).toBe('lemonjoe');
  });

  it('signIn failure sets the error and stays signed out', async () => {
    const done = store.signIn('lemonjoe');
    http
      .expectOne('/api/login')
      .flush(
        { error: 'account_secured', message: 'This username is protected.' },
        { status: 409, statusText: '' },
      );

    expect(await done).toBeFalse();
    expect(store.error()).toBe('This username is protected.');
    expect(session.username()).toBeNull();
  });

  it('load stores the game view', async () => {
    const done = store.load();
    http.expectOne('/api/game').flush(newGameView());
    await done;

    expect(store.game()?.capital).toBe(1000);
    expect(store.loading()).toBeFalse();
  });

  it('a mutation replaces the game view with the response', async () => {
    const done = store.buy('lemon', 2);
    http.expectOne('/api/game/buy').flush(newGameView({ capital: 956 }));
    await done;

    expect(store.game()?.capital).toBe(956);
  });

  it('passes clamp through to the API', async () => {
    const done = store.sell('lemon', 50, true);
    const req = http.expectOne('/api/game/sell');
    expect(req.request.body).toEqual({ resource: 'lemon', qty: 50, clamp: true });
    req.flush(newGameView());
    await done;
  });

  it('a failed mutation keeps the game and shows the server message', async () => {
    const load = store.load();
    http.expectOne('/api/game').flush(newGameView());
    await load;

    const done = store.sell('lemonade', 1);
    http
      .expectOne('/api/game/sell')
      .flush(
        { error: 'insufficient_stock', message: 'Not enough lemonade' },
        { status: 409, statusText: '' },
      );
    await done;

    expect(store.error()).toBe('Not enough lemonade');
    expect(store.game()?.capital).toBe(1000);
  });

  it('a successful call clears the previous error', async () => {
    const fail = store.buy('lemon', 1);
    http
      .expectOne('/api/game/buy')
      .flush({ error: 'x', message: 'nope' }, { status: 409, statusText: '' });
    await fail;

    const ok = store.buy('lemon', 1);
    http.expectOne('/api/game/buy').flush(newGameView());
    await ok;

    expect(store.error()).toBeNull();
  });

  it('endDay stores the report and the new game view', async () => {
    const done = store.endDay();
    http
      .expectOne('/api/game/end-day')
      .flush({ report: dayReport(), game: newGameView({ day: 5 }) });
    await done;

    expect(store.report()?.day).toBe(4);
    expect(store.game()?.day).toBe(5);

    store.dismissReport();
    expect(store.report()).toBeNull();
  });

  it('isBankrupt follows the game status', async () => {
    const done = store.load();
    http.expectOne('/api/game').flush(newGameView({ status: 'bankrupt' }));
    await done;

    expect(store.isBankrupt()).toBeTrue();
  });

  it('signOut signs out of Firebase and clears the session and the game', async () => {
    session.signIn('lemonjoe');
    const done = store.load();
    http.expectOne('/api/game').flush(newGameView());
    await done;

    store.signOut();
    expect(port.signOutCalls).toBe(1);
    expect(session.username()).toBeNull();
    expect(store.game()).toBeNull();
  });

  it('loading is true while a request is in flight and false after it succeeds or fails', async () => {
    const ok = store.load();
    expect(store.loading()).toBeTrue();
    http.expectOne('/api/game').flush(newGameView());
    await ok;
    expect(store.loading()).toBeFalse();

    const failed = store.load();
    expect(store.loading()).toBeTrue();
    http
      .expectOne('/api/game')
      .flush({ error: 'internal', message: 'Boom' }, { status: 500, statusText: '' });
    await failed;
    expect(store.loading()).toBeFalse();
  });

  it('a failed end-of-day leaves the game and the previous report alone', async () => {
    const first = store.endDay();
    http
      .expectOne('/api/game/end-day')
      .flush({ report: dayReport({ day: 1 }), game: newGameView({ day: 2 }) });
    await first;

    const second = store.endDay();
    http
      .expectOne('/api/game/end-day')
      .flush({ error: 'game_over', message: 'Game over' }, { status: 409, statusText: '' });
    await second;

    expect(store.error()).toBe('Game over');
    expect(store.report()?.day).toBe(1);
    expect(store.game()?.day).toBe(2);
  });

  it('starting a new game clears the last day report', async () => {
    const day = store.endDay();
    http.expectOne('/api/game/end-day').flush({ report: dayReport(), game: newGameView() });
    await day;
    expect(store.report()).not.toBeNull();

    const fresh = store.newGame();
    expect(store.report()).toBeNull();
    http.expectOne('/api/game/new').flush(newGameView());
    await fresh;
  });

  it('dismissReport and clearError reset their signals', async () => {
    const day = store.endDay();
    http.expectOne('/api/game/end-day').flush({ report: dayReport(), game: newGameView() });
    await day;
    store.dismissReport();
    expect(store.report()).toBeNull();

    const failed = store.load();
    http
      .expectOne('/api/game')
      .flush({ error: 'internal', message: 'Boom' }, { status: 500, statusText: '' });
    await failed;
    expect(store.error()).toBe('Boom');
    store.clearError();
    expect(store.error()).toBeNull();
  });

  it('toasts each achievement a mutation unlocked, once', async () => {
    const toasts = TestBed.inject(ToastService);
    const done = store.buy('lemon', 1);
    http
      .expectOne('/api/game/buy')
      .flush(newGameView({ unlocked: [{ key: 'first_expand', name: 'Growing', tier: 'bronze' }] }));
    await done;

    expect(toasts.toasts().map((t) => t.text)).toEqual(['Achievement unlocked: Growing']);

    const load = store.load();
    http.expectOne('/api/game').flush(newGameView());
    await load;
    expect(toasts.toasts().length).toBe(1);
  });

  it('toasts the achievements an end of day unlocked', async () => {
    const toasts = TestBed.inject(ToastService);
    const done = store.endDay();
    http.expectOne('/api/game/end-day').flush({
      report: dayReport(),
      game: newGameView({ unlocked: [{ key: 'day_7', name: 'First week', tier: 'bronze' }] }),
    });
    await done;

    expect(toasts.toasts().map((t) => t.text)).toEqual(['Achievement unlocked: First week']);
  });
});
