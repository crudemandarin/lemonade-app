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
    ['login', () => api.login('lemonjoe'), 'POST', '/api/login', { username: 'lemonjoe' }],
    ['getGame', () => api.getGame(), 'GET', '/api/game', null],
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
