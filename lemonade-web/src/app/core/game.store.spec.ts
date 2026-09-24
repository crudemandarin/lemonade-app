import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { GameStore } from './game.store';
import { SessionService } from './session.service';
import { dayReport, newGameView } from './testing/fixtures';

describe('GameStore', () => {
  let store: GameStore;
  let http: HttpTestingController;
  let session: SessionService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
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

  it('signIn logs in, stores the username, and resolves true', async () => {
    const done = store.signIn('lemonjoe');
    http.expectOne('/api/login').flush({ id: 1, username: 'lemonjoe' });

    expect(await done).toBeTrue();
    expect(session.username()).toBe('lemonjoe');
  });

  it('signIn failure sets the error and stays signed out', async () => {
    const done = store.signIn('x');
    http
      .expectOne('/api/login')
      .flush({ error: 'invalid_username', message: 'Bad name' }, { status: 400, statusText: '' });

    expect(await done).toBeFalse();
    expect(store.error()).toBe('Bad name');
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

  it('signOut clears the session and the game', async () => {
    session.signIn('lemonjoe');
    const done = store.load();
    http.expectOne('/api/game').flush(newGameView());
    await done;

    store.signOut();
    expect(session.username()).toBeNull();
    expect(store.game()).toBeNull();
  });
});
