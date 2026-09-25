import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

import { EndDayResponse, FacilityType, GameView, Resource, User } from './api.models';

const API_URL = '/api';

/** Typed HTTP client for lemonade-api. Only GameStore should use it. */
@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly http = inject(HttpClient);

  login(username: string): Observable<User> {
    return this.http.post<User>(`${API_URL}/login`, { username });
  }

  getGame(): Observable<GameView> {
    return this.http.get<GameView>(`${API_URL}/game`);
  }

  newGame(): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/new`, {});
  }

  /** With `clamp`, the server trades as many as it can up to `qty` instead of failing. */
  buy(resource: Resource, qty: number, clamp = false): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/buy`, {
      resource,
      qty,
      ...(clamp && { clamp }),
    });
  }

  sell(resource: Resource, qty: number, clamp = false): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/sell`, {
      resource,
      qty,
      ...(clamp && { clamp }),
    });
  }

  expandWarehouse(resource: Resource): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/facilities/warehouse/expand`, { resource });
  }

  expandProduction(): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/facilities/production/expand`, {});
  }

  upgrade(type: FacilityType): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/facilities/${type}/upgrade`, {});
  }

  endDay(): Observable<EndDayResponse> {
    return this.http.post<EndDayResponse>(`${API_URL}/game/end-day`, {});
  }
}
