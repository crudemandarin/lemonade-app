import { Component, computed, input, signal } from '@angular/core';

import { formatMoney } from '../money.pipe';

const WIDTH = 88;
const HEIGHT = 28;
const PAD_X = 5; // room for the end dot and its ring
const PAD_Y = 6;

/**
 * A tiny price-history line. It is deliberately quiet: a muted line, one ink dot on
 * today's price. The y range is its own (min to max of the history), because the point
 * is direction, not comparison between resources. Hovering shows the price for a day.
 */
@Component({
  selector: 'app-price-sparkline',
  standalone: true,
  templateUrl: './price-sparkline.component.html',
  styleUrl: './price-sparkline.component.scss',
})
export class PriceSparklineComponent {
  /** Effective prices, oldest first. */
  readonly values = input.required<number[]>();
  /** What is being drawn, for the accessible description. */
  readonly label = input('Price');

  protected readonly width = WIDTH;
  protected readonly height = HEIGHT;
  protected readonly hover = signal<number | null>(null);

  protected readonly points = computed(() => {
    const v = this.values();
    const min = Math.min(...v);
    const max = Math.max(...v);
    const span = max - min;
    return v.map((value, i) => ({
      x: v.length === 1 ? WIDTH - PAD_X : PAD_X + (i * (WIDTH - 2 * PAD_X)) / (v.length - 1),
      // A flat history sits in the middle instead of dividing by zero.
      y: span === 0 ? HEIGHT / 2 : PAD_Y + ((max - value) * (HEIGHT - 2 * PAD_Y)) / span,
      value,
    }));
  });

  protected readonly path = computed(() => {
    const pts = this.points();
    return pts.length < 2
      ? ''
      : pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)} ${p.y.toFixed(1)}`).join('');
  });

  protected readonly last = computed(() => this.points()[this.points().length - 1]);

  protected readonly shown = computed(() => {
    const i = this.hover();
    return i === null ? null : this.points()[i];
  });

  protected readonly tip = computed(() => {
    const i = this.hover();
    if (i === null) {
      return '';
    }
    const ago = this.points().length - 1 - i;
    const when = ago === 0 ? 'today' : ago === 1 ? 'yesterday' : `${ago} days ago`;
    return `${formatMoney(this.points()[i].value)} ${when}`;
  });

  protected readonly description = computed(() => {
    const v = this.values();
    const days = v.length === 1 ? '1 day' : `${v.length} days`;
    return v.length < 2
      ? `${this.label()} price today: ${formatMoney(v[0])}`
      : `${this.label()} price over ${days}: from ${formatMoney(v[0])} to ${formatMoney(v[v.length - 1])}`;
  });

  protected onMove(event: PointerEvent): void {
    const box = (event.currentTarget as SVGElement).getBoundingClientRect();
    const pts = this.points();
    if (pts.length < 2 || box.width === 0) {
      return;
    }
    const x = ((event.clientX - box.left) / box.width) * WIDTH;
    let best = 0;
    for (let i = 1; i < pts.length; i++) {
      if (Math.abs(pts[i].x - x) < Math.abs(pts[best].x - x)) {
        best = i;
      }
    }
    this.hover.set(best);
  }
}
