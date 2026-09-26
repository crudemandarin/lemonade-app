import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

import {
  AchievementsResponse,
  Board,
  DayReport,
  EndDayResponse,
  FacilityType,
  GameView,
  ReportSummary,
  Resource,
  RunDetail,
  RunSummary,
  ScoresResponse,
  UpgradesResponse,
  EmpireResponse,
  User,
} from './api.models';

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

  /** Adds a building to a storage class's pool. */
  expandWarehouse(storageClass: string): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/facilities/warehouse/expand`, {
      class: storageClass,
    });
  }

  expandProduction(): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/facilities/production/expand`, {});
  }

  /** Sells one building; `storageClass` picks the warehouse pool (ignored for production). */
  sellFacility(type: FacilityType, storageClass?: string): Observable<GameView> {
    return this.http.post<GameView>(
      `${API_URL}/game/facilities/${type}/sell`,
      type === 'warehouse' ? { class: storageClass } : {},
    );
  }

  upgrade(type: FacilityType): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/facilities/${type}/upgrade`, {});
  }

  /** Ends the run; the server records its score. */
  giveUp(): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/give-up`, {});
  }

  /** Ended days of the current run (or of a finished run of this user), oldest first. */
  listReports(runId?: string): Observable<ReportSummary[]> {
    return this.http.get<ReportSummary[]>(`${API_URL}/game/reports`, {
      params: runId ? { runId } : {},
    });
  }

  getReport(day: number, runId?: string): Observable<DayReport> {
    return this.http.get<DayReport>(`${API_URL}/game/reports/${day}`, {
      params: runId ? { runId } : {},
    });
  }

  /** The global board: each player's best run. `me` is the caller's own row. */
  scores(limit?: number, board: Board = 'all_time'): Observable<ScoresResponse> {
    return this.http.get<ScoresResponse>(`${API_URL}/scores`, {
      params: {
        ...(limit && { limit }),
        ...(board !== 'all_time' && { board }),
      },
    });
  }

  /** Every achievement with the caller's unlocked state and progress. */
  achievements(): Observable<AchievementsResponse> {
    return this.http.get<AchievementsResponse>(`${API_URL}/achievements`);
  }

  /** The caller's finished runs, newest first. */
  runs(): Observable<RunSummary[]> {
    return this.http.get<RunSummary[]>(`${API_URL}/runs`);
  }

  /** One of the caller's finished runs in full. */
  run(runId: string): Observable<RunDetail> {
    return this.http.get<RunDetail>(`${API_URL}/runs/${encodeURIComponent(runId)}`);
  }

  /** Every upgrade with its state (owned, available, locked and why). */
  upgrades(): Observable<UpgradesResponse> {
    return this.http.get<UpgradesResponse>(`${API_URL}/game/upgrades`);
  }

  /** Buys one upgrade; answers with the new game view. */
  buyUpgrade(key: string): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/upgrades/${encodeURIComponent(key)}/buy`, {});
  }

  /** Every territory with its share, rivals, telegraphs and prices. */
  empire(): Observable<EmpireResponse> {
    return this.http.get<EmpireResponse>(`${API_URL}/game/empire`);
  }

  enterTerritory(key: string): Observable<GameView> {
    return this.http.post<GameView>(
      `${API_URL}/game/territories/${encodeURIComponent(key)}/enter`,
      {},
    );
  }

  campaign(key: string, level: number): Observable<GameView> {
    return this.http.post<GameView>(
      `${API_URL}/game/territories/${encodeURIComponent(key)}/campaign`,
      {
        level,
      },
    );
  }

  buyOut(key: string, hostile: boolean): Observable<GameView> {
    return this.http.post<GameView>(`${API_URL}/game/rivals/${encodeURIComponent(key)}/buyout`, {
      hostile,
    });
  }

  endDay(): Observable<EndDayResponse> {
    return this.http.post<EndDayResponse>(`${API_URL}/game/end-day`, {});
  }
}
