import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Observable } from 'rxjs';

import { ApiService } from './api.service';

describe('ApiService', () => {
  let api: ApiService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    api = TestBed.inject(ApiService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  const cases: [string, () => Observable<unknown>, string, string, unknown][] = [
    ['me', () => api.me(), 'GET', '/api/me', null],
    [
      'createProfile',
      () => api.createProfile('lemonjoe'),
      'POST',
      '/api/me/username',
      { username: 'lemonjoe' },
    ],
    ['claim', () => api.claim('oldtimer'), 'POST', '/api/me/claim', { username: 'oldtimer' }],
    ['getGame', () => api.getGame(), 'GET', '/api/game', null],
    ['scores', () => api.scores(), 'GET', '/api/scores', null],
    ['scores with a limit', () => api.scores(5), 'GET', '/api/scores?limit=5', null],
    ['runs', () => api.runs(), 'GET', '/api/runs', null],
    ['run', () => api.run('abc-123'), 'GET', '/api/runs/abc-123', null],
    ['newGame', () => api.newGame(), 'POST', '/api/game/new', {}],
    ['buy', () => api.buy('lemon', 3), 'POST', '/api/game/buy', { resource: 'lemon', qty: 3 }],
    [
      'sell',
      () => api.sell('lemonade', 2),
      'POST',
      '/api/game/sell',
      { resource: 'lemonade', qty: 2 },
    ],
    [
      'clamped buy',
      () => api.buy('lemon', 100, true),
      'POST',
      '/api/game/buy',
      { resource: 'lemon', qty: 100, clamp: true },
    ],
    [
      'clamped sell',
      () => api.sell('lemon', 100, true),
      'POST',
      '/api/game/sell',
      { resource: 'lemon', qty: 100, clamp: true },
    ],
    [
      'sell a warehouse',
      () => api.sellFacility('warehouse', 'ice'),
      'POST',
      '/api/game/facilities/warehouse/sell',
      { resource: 'ice' },
    ],
    [
      'sell a production building',
      () => api.sellFacility('production'),
      'POST',
      '/api/game/facilities/production/sell',
      {},
    ],
    [
      'expandWarehouse',
      () => api.expandWarehouse('ice'),
      'POST',
      '/api/game/facilities/warehouse/expand',
      { resource: 'ice' },
    ],
    [
      'expandProduction',
      () => api.expandProduction(),
      'POST',
      '/api/game/facilities/production/expand',
      {},
    ],
    [
      'upgrade',
      () => api.upgrade('warehouse'),
      'POST',
      '/api/game/facilities/warehouse/upgrade',
      {},
    ],
    ['endDay', () => api.endDay(), 'POST', '/api/game/end-day', {}],
  ];

  for (const [name, call, method, url, body] of cases) {
    it(`${name} sends ${method} ${url}`, () => {
      call().subscribe();
      const req = http.expectOne({ method, url });
      expect(req.request.body).toEqual(body);
      req.flush({});
    });
  }
});
