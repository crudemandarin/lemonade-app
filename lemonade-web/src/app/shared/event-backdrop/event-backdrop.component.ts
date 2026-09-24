import { Component, computed, input } from '@angular/core';

import { GameEvent } from '../../core/api.models';

type Scene =
  'heat_wave' | 'rainy_week' | 'lemon_blight' | 'sugar_glut' | 'holiday' | 'cup_shortage';

const SCENES: readonly string[] = [
  'heat_wave',
  'rainy_week',
  'lemon_blight',
  'sugar_glut',
  'holiday',
  'cup_shortage',
];

/** Most scenes drawn at once; more would compete with the foreground. */
const MAX_SCENES = 2;

/** Particle counts per scene. Kept low: the layer covers the whole viewport. */
const COUNTS = {
  drops: 36,
  leaves: 12,
  sugar: 12,
  confetti: 32,
  cups: 10,
  bunting: 12,
  shimmer: 3,
};

/**
 * Looping, low-detail background for the active random events. Fixed behind the
 * page, covers the viewport, ignores pointer input, and is decorative only.
 * Particles get deterministic positions and negative delays, so each scene is
 * already mid-loop when it appears.
 */
@Component({
  selector: 'app-event-backdrop',
  standalone: true,
  templateUrl: './event-backdrop.component.html',
  styleUrl: './event-backdrop.component.scss',
})
export class EventBackdropComponent {
  readonly events = input<GameEvent[]>([]);

  /** Known events only, longest-running first, capped at MAX_SCENES. */
  protected readonly scenes = computed(() =>
    [...this.events()]
      .filter((e) => SCENES.includes(e.key))
      .sort((a, b) => b.daysLeft - a.daysLeft)
      .slice(0, MAX_SCENES)
      .map((e) => e.key as Scene),
  );

  protected readonly drops = range(COUNTS.drops);
  protected readonly leaves = range(COUNTS.leaves);
  protected readonly sugar = range(COUNTS.sugar);
  protected readonly confetti = range(COUNTS.confetti);
  protected readonly cups = range(COUNTS.cups);
  protected readonly bunting = range(COUNTS.bunting);
  protected readonly shimmer = range(COUNTS.shimmer);

  /**
   * CSS variables for particle `i` of `n`: x position (%), animation duration and
   * a negative delay (s), and a 0-1 jitter value. `spread` is the duration range.
   */
  protected vars(i: number, n: number, minDur: number, spread: number): Record<string, string> {
    const jitter = (i * 0.6180339887) % 1;
    const dur = minDur + ((i * 0.3819660113) % 1) * spread;
    return {
      '--x': `${((i + 0.5) / n) * 100}%`,
      '--dur': `${dur.toFixed(2)}s`,
      '--delay': `${(-jitter * dur).toFixed(2)}s`,
      '--j': jitter.toFixed(3),
    };
  }
}

function range(n: number): number[] {
  return Array.from({ length: n }, (_, i) => i);
}
