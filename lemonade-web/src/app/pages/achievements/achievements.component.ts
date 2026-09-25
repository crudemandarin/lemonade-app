import { DatePipe } from '@angular/common';
import { Component, OnInit, computed, inject, signal } from '@angular/core';

import { Achievement, AchievementsResponse } from '../../core/api.models';
import { ScoresService } from '../../core/scores.service';
import { CardComponent } from '../../shared/card/card.component';

type Load = 'loading' | 'ready' | 'error';

/**
 * Every achievement, grouped by category. Tiers are written out (not just coloured),
 * measurable goals show a progress bar with its numbers, and hidden ones read "???"
 * until they unlock. Achievements are cosmetic: they never change how the game plays.
 */
@Component({
  selector: 'app-achievements',
  standalone: true,
  imports: [CardComponent, DatePipe],
  templateUrl: './achievements.component.html',
  styleUrl: './achievements.component.scss',
})
export class AchievementsComponent implements OnInit {
  private readonly scores = inject(ScoresService);

  protected readonly state = signal<Load>('loading');
  protected readonly data = signal<AchievementsResponse | null>(null);

  /** Categories that have rows, each with its rows in table order. */
  protected readonly groups = computed(() => {
    const d = this.data();
    if (!d) {
      return [];
    }
    return d.categories
      .map((c) => ({ ...c, rows: d.achievements.filter((a) => a.category === c.key) }))
      .filter((g) => g.rows.length > 0);
  });

  ngOnInit(): void {
    this.load();
  }

  protected async load(): Promise<void> {
    this.state.set('loading');
    try {
      this.data.set(await this.scores.achievements());
      this.state.set('ready');
    } catch {
      this.state.set('error');
    }
  }

  protected percent(a: Achievement): number {
    return a.progress ? Math.round((100 * a.progress.current) / a.progress.target) : 0;
  }
}
