import { DatePipe } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { RunSummary, ScoreRow } from '../../core/api.models';
import { ScoresService } from '../../core/scores.service';
import { CardComponent } from '../../shared/card/card.component';
import { MoneyPipe } from '../../shared/money.pipe';

type Tab = 'global' | 'mine';
type Load = 'loading' | 'ready' | 'error';

/**
 * High scores. Global lists each player's best run; Mine is the player's own record
 * history, and each of its rows opens that run in full. Boards live in this page's
 * own state, not in GameStore.
 */
@Component({
  selector: 'app-scores',
  standalone: true,
  imports: [CardComponent, DatePipe, MoneyPipe, RouterLink],
  templateUrl: './scores.component.html',
  styleUrl: './scores.component.scss',
})
export class ScoresComponent implements OnInit {
  private readonly scores = inject(ScoresService);

  protected readonly tab = signal<Tab>('global');

  protected readonly globalState = signal<Load>('loading');
  protected readonly rows = signal<ScoreRow[]>([]);
  /** The caller's own row when it is below the rows shown. */
  protected readonly meBelow = signal<ScoreRow | null>(null);

  protected readonly mineState = signal<Load>('loading');
  protected readonly runs = signal<RunSummary[]>([]);

  ngOnInit(): void {
    this.loadGlobal();
  }

  protected select(tab: Tab): void {
    this.tab.set(tab);
    if (tab === 'global') {
      this.loadGlobal();
    } else {
      this.loadMine();
    }
  }

  protected async loadGlobal(): Promise<void> {
    this.globalState.set('loading');
    try {
      const board = await this.scores.scores();
      this.rows.set(board.rows);
      const inRows = board.me !== null && board.rows.some((r) => r.isMe);
      this.meBelow.set(board.me && !inRows ? board.me : null);
      this.globalState.set('ready');
    } catch {
      this.globalState.set('error');
    }
  }

  protected async loadMine(): Promise<void> {
    this.mineState.set('loading');
    try {
      this.runs.set(await this.scores.myRuns());
      this.mineState.set('ready');
    } catch {
      this.mineState.set('error');
    }
  }

  protected ending(run: RunSummary): string {
    return run.endedBy === 'gave_up' ? 'Gave up' : 'Went bankrupt';
  }
}
