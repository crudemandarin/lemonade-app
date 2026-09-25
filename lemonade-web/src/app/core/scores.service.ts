import { Injectable, inject } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import {
  AchievementsResponse,
  DayReport,
  RunDetail,
  RunSummary,
  ScoresResponse,
} from './api.models';
import { ApiService } from './api.service';

/**
 * Reads the global board and the player's record history. These live in their own
 * page state, not in GameStore, which owns the current game only. Every method
 * rejects with the HTTP error on failure.
 */
@Injectable({ providedIn: 'root' })
export class ScoresService {
  private readonly api = inject(ApiService);

  scores(limit?: number): Promise<ScoresResponse> {
    return firstValueFrom(this.api.scores(limit));
  }

  achievements(): Promise<AchievementsResponse> {
    return firstValueFrom(this.api.achievements());
  }

  myRuns(): Promise<RunSummary[]> {
    return firstValueFrom(this.api.runs());
  }

  run(runId: string): Promise<RunDetail> {
    return firstValueFrom(this.api.run(runId));
  }

  /** One day's report from a finished run of this player. */
  report(runId: string, day: number): Promise<DayReport> {
    return firstValueFrom(this.api.getReport(day, runId));
  }
}
