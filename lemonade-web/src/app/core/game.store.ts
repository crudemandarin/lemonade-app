import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, firstValueFrom } from 'rxjs';

import { apiErrorMessage } from './api-error';
import {
  DayReport,
  FacilityType,
  GameView,
  ReportSummary,
  Resource,
  UpgradesResponse,
} from './api.models';
import { ApiService } from './api.service';
import { SessionService } from './session.service';
import { ToastService } from './toast.service';

/**
 * The single source of game state in the UI. Every mutation returns the updated
 * game view from the server, so the store never recomputes game outcomes itself.
 */
@Injectable({ providedIn: 'root' })
export class GameStore {
  private readonly api = inject(ApiService);
  private readonly session = inject(SessionService);
  private readonly toasts = inject(ToastService);

  private readonly _game = signal<GameView | null>(null);
  private readonly _report = signal<DayReport | null>(null);
  private readonly _error = signal<string | null>(null);
  private readonly _loading = signal(false);

  readonly game = this._game.asReadonly();
  /** The last end-of-day report, until the player dismisses it. */
  readonly report = this._report.asReadonly();
  /** Message from the last failed request, cleared by the next successful one. */
  readonly error = this._error.asReadonly();
  readonly loading = this._loading.asReadonly();
  readonly isBankrupt = computed(() => this._game()?.status === 'bankrupt');

  async signIn(username: string): Promise<boolean> {
    const user = await this.run(this.api.login(username));
    if (!user) {
      return false;
    }
    this._game.set(null);
    this.session.signIn(user.username);
    return true;
  }

  signOut(): void {
    this.session.signOut();
    this._game.set(null);
    this._report.set(null);
    this._error.set(null);
  }

  load(): Promise<void> {
    return this.update(this.api.getGame());
  }

  newGame(): Promise<void> {
    this._report.set(null);
    return this.update(this.api.newGame());
  }

  buy(resource: Resource, qty: number, clamp = false): Promise<void> {
    return this.update(this.api.buy(resource, qty, clamp));
  }

  sell(resource: Resource, qty: number, clamp = false): Promise<void> {
    return this.update(this.api.sell(resource, qty, clamp));
  }

  expandWarehouse(resource: Resource): Promise<void> {
    return this.update(this.api.expandWarehouse(resource));
  }

  expandProduction(): Promise<void> {
    return this.update(this.api.expandProduction());
  }

  sellFacility(type: FacilityType, resource?: Resource): Promise<void> {
    return this.update(this.api.sellFacility(type, resource));
  }

  upgrade(type: FacilityType): Promise<void> {
    return this.update(this.api.upgrade(type));
  }

  /** The upgrade list is read on demand and kept out of the game state. Rejects on failure. */
  upgradeList(): Promise<UpgradesResponse> {
    return firstValueFrom(this.api.upgrades());
  }

  /** Buys an upgrade; the game view (cash, features, upkeep) comes back with it. */
  buyUpgrade(key: string): Promise<void> {
    return this.update(this.api.buyUpgrade(key));
  }

  /** Past days are read on demand and kept out of the game state. Rejects on failure. */
  reportSummaries(): Promise<ReportSummary[]> {
    return firstValueFrom(this.api.listReports());
  }

  reportForDay(day: number): Promise<DayReport> {
    return firstValueFrom(this.api.getReport(day));
  }

  giveUp(): Promise<void> {
    return this.update(this.api.giveUp());
  }

  async endDay(): Promise<void> {
    const res = await this.run(this.api.endDay());
    if (res) {
      this._game.set(res.game);
      this._report.set(res.report);
      this.toasts.unlocked(res.game.unlocked);
    }
  }

  dismissReport(): void {
    this._report.set(null);
  }

  clearError(): void {
    this._error.set(null);
  }

  private async update(request: Observable<GameView>): Promise<void> {
    const game = await this.run(request);
    if (game) {
      this._game.set(game);
      this.toasts.unlocked(game.unlocked);
    }
  }

  /** Runs one request, recording loading and error state. Resolves undefined on failure. */
  private async run<T>(request: Observable<T>): Promise<T | undefined> {
    this._loading.set(true);
    try {
      const result = await firstValueFrom(request);
      this._error.set(null);
      return result;
    } catch (err) {
      this._error.set(apiErrorMessage(err));
      return undefined;
    } finally {
      this._loading.set(false);
    }
  }
}
