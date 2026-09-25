import {
  Component,
  DestroyRef,
  ElementRef,
  afterNextRender,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';

import { PricePoint, Resource, TimelinePoint } from '../../core/api.models';
import { RESOURCE_LABELS, RESOURCE_ORDER } from '../../core/resources';
import { formatMoney } from '../money.pipe';
import {
  describePoint,
  dayTicks,
  eventBands,
  PriceMode,
  priceAt,
  priceSeries,
  formatCompactMoney,
  markerPoints,
  nearestIndex,
  niceTicks,
  placePoints,
  stepPath,
} from './timeline-layout';

/** Time-range choices for the three charts, in game days back from the latest point. */
export const RANGES: { label: string; days: number | null }[] = [
  { label: '14d', days: 14 },
  { label: '1m', days: 30 },
  { label: '3m', days: 90 },
  { label: '6m', days: 180 },
  { label: 'All', days: null },
];

let nextChartId = 0;

const PHONE = 480; // below this width, margins shrink and the latest-value label moves inside
const TOP = 10;
const BOTTOM = 6;
const X_BAND = 24; // day labels under the lower chart

/**
 * Three charts on one shared time axis: capital (with a dot for every trade and facility
 * purchase), the stock of each resource, and the price of each resource. Never one
 * dual-axis plot: capital is dollars in the thousands, stock is cases in the tens.
 * Pure SVG; the layout math lives in timeline-layout.ts.
 */
@Component({
  selector: 'app-timeline-charts',
  standalone: true,
  templateUrl: './timeline-charts.component.html',
  styleUrl: './timeline-charts.component.scss',
})
export class TimelineChartsComponent {
  /** Oldest first, straight from the game view. */
  readonly points = input.required<TimelinePoint[]>();
  readonly size = input<'compact' | 'large'>('compact');
  /** One point per day; the price chart starts wherever the log does. */
  readonly priceLog = input<PricePoint[]>([]);
  /** Long-run prices, for the percent view. */
  readonly basePrices = input<number[]>([]);

  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly width = signal(640);
  protected readonly viewTable = signal(false);
  protected readonly hidden = signal<ReadonlySet<Resource>>(new Set());
  protected readonly hoverIndex = signal<number | null>(null);
  protected readonly hoverChart = signal<'capital' | 'stock' | 'price' | null>(null);
  protected readonly priceMode = signal<PriceMode>('dollars');
  protected readonly priceHidden = signal<ReadonlySet<Resource>>(new Set());

  protected readonly resources = RESOURCE_ORDER;
  protected readonly labels = RESOURCE_LABELS;
  protected readonly formatMoney = formatMoney;
  protected readonly describe = describePoint;

  constructor() {
    afterNextRender(() => {
      const el = this.host.nativeElement;
      const measure = () => this.width.set(Math.max(300, el.clientWidth || 640));
      measure();
      const observer = new ResizeObserver(measure);
      observer.observe(el);
      this.destroyRef.onDestroy(() => observer.disconnect());
    });
  }

  // ---- layout -------------------------------------------------------------------

  protected readonly capHeight = computed(() => (this.size() === 'large' ? 220 : 150));
  protected readonly stockHeight = computed(() => (this.size() === 'large' ? 190 : 130) + X_BAND);
  protected readonly phone = computed(() => this.width() < PHONE);
  // Room for y-axis labels on the left, and (on wider screens) the latest-value label on the right.
  protected readonly left = computed(() => (this.phone() ? 40 : 48));
  private readonly right = computed(() => (this.phone() ? 14 : 64));
  protected readonly plotWidth = computed(() =>
    Math.max(60, this.width() - this.left() - this.right()),
  );

  protected readonly placed = computed(() => placePoints(this.points()));
  private readonly dataMin = computed(() => this.placed()[0]?.x ?? 1);
  private readonly xMax = computed(() => {
    const p = this.placed();
    return Math.max(p[p.length - 1]?.x ?? 2, this.dataMin() + 1);
  });

  /** The time range shown, in game days back from now; null shows everything. */
  protected readonly range = signal<number | null>(null);
  protected readonly ranges = RANGES;
  /** Left edge of the axis: the start of the game, or `range` days before the latest point. */
  private readonly xMin = computed(() => {
    const days = this.range();
    return days === null ? this.dataMin() : Math.max(this.dataMin(), this.xMax() - days);
  });

  /**
   * Index of the first point that matters for scaling the axes: the last one at or before
   * the window's left edge (its value is still in force there), or 0 when showing everything.
   */
  private readonly windowStart = computed(() => {
    const placed = this.placed();
    const first = placed.findIndex((p) => p.x >= this.xMin());
    return first === -1 ? Math.max(0, placed.length - 1) : Math.max(0, first - 1);
  });
  /** Index of the first point inside the window; the earliest a cursor or key can reach. */
  private readonly firstShown = computed(() => {
    const first = this.placed().findIndex((p) => p.x >= this.xMin());
    return first === -1 ? Math.max(0, this.placed().length - 1) : first;
  });
  private readonly clipId = `tl-clip-${nextChartId++}`;
  protected clipUrl(chart: 'cap' | 'stock' | 'price'): string {
    return `url(#${this.clipId}-${chart})`;
  }
  protected clipName(chart: 'cap' | 'stock' | 'price'): string {
    return `${this.clipId}-${chart}`;
  }

  protected readonly enough = computed(() => this.placed().length >= 2);

  protected xPx(x: number): number {
    return this.left() + ((x - this.xMin()) / (this.xMax() - this.xMin())) * this.plotWidth();
  }

  protected readonly xs = computed(() => this.placed().map((p) => this.xPx(p.x)));

  protected readonly dayGrid = computed(() => {
    const first = Math.floor(this.xMin());
    const last = Math.floor(this.xMax());
    const ticks = dayTicks(first, last, Math.max(2, Math.floor(this.plotWidth() / 48)));
    return ticks.map((day) => ({ day, x: this.xPx(day) })).filter((t) => t.x >= this.left() - 0.5);
  });

  // ---- capital chart --------------------------------------------------------------

  private readonly capTicks = computed(() => {
    const caps = this.placed()
      .slice(this.windowStart())
      .map((p) => p.point.capital);
    return niceTicks(Math.min(0, ...caps), Math.max(...caps, 1), 3);
  });

  private capY(v: number): number {
    const t = this.capTicks();
    const min = t[0];
    const max = t[t.length - 1];
    return TOP + ((max - v) / (max - min)) * (this.capHeight() - TOP - BOTTOM);
  }

  protected readonly capGrid = computed(() =>
    this.capTicks().map((v) => ({ y: this.capY(v), label: formatCompactMoney(v) })),
  );

  protected readonly capLine = computed(() =>
    stepPath(
      this.xs(),
      this.placed().map((p) => this.capY(p.point.capital)),
    ),
  );

  protected readonly capArea = computed(() => {
    const xs = this.xs();
    if (xs.length < 2) {
      return '';
    }
    const base = this.capY(this.capTicks()[0]);
    return `${this.capLine()}V${base}H${xs[0]}Z`;
  });

  protected readonly capMarkers = computed(() =>
    markerPoints(this.placed())
      .filter((p) => p.x >= this.xMin())
      .map((p) => ({
        index: p.index,
        kind: p.point.kind,
        x: this.xPx(p.x),
        y: this.capY(p.point.capital),
      })),
  );

  protected readonly capEnd = computed(() => {
    const last = this.placed()[this.placed().length - 1];
    if (!last) {
      return null;
    }
    const value = last.point.capital;
    return {
      x: this.xPx(last.x),
      y: this.capY(value),
      label: value >= 100_000 ? formatCompactMoney(value) : formatMoney(value),
    };
  });

  protected readonly capSummary = computed(() => {
    const p = this.points();
    if (p.length === 0) {
      return 'Capital over time';
    }
    const peak = Math.max(...p.map((x) => x.capital));
    return `Capital over time: started at ${formatMoney(p[0].capital)}, now ${formatMoney(p[p.length - 1].capital)}, peak ${formatMoney(peak)}. Use the table view for every value.`;
  });

  // ---- stock chart ----------------------------------------------------------------

  private readonly visible = computed(() => this.resources.filter((r) => !this.hidden().has(r)));

  private readonly stockTicks = computed(() => {
    const idx = this.visible().map((r) => RESOURCE_ORDER.indexOf(r));
    // At least 4, so the ticks are whole cases even when almost nothing is in stock.
    const shown = this.points().slice(this.windowStart());
    const max = Math.max(4, ...shown.flatMap((p) => idx.map((i) => p.stock[i] ?? 0)));
    return niceTicks(0, max, 3);
  });

  private stockPlotHeight(): number {
    return this.stockHeight() - X_BAND;
  }

  private stockY(v: number): number {
    const t = this.stockTicks();
    return (
      TOP +
      ((t[t.length - 1] - v) / (t[t.length - 1] - t[0])) * (this.stockPlotHeight() - TOP - BOTTOM)
    );
  }

  protected readonly stockGrid = computed(() =>
    this.stockTicks().map((v) => ({ y: this.stockY(v), label: String(v) })),
  );

  protected readonly stockLines = computed(() =>
    this.visible().map((resource) => {
      const i = RESOURCE_ORDER.indexOf(resource);
      return {
        resource,
        color: `var(--series-${resource})`,
        d: stepPath(
          this.xs(),
          this.points().map((p) => this.stockY(p.stock[i] ?? 0)),
        ),
      };
    }),
  );

  protected readonly stockAxisY = computed(() => this.stockPlotHeight());

  // ---- price chart ----------------------------------------------------------------

  private readonly priceVisible = computed(() =>
    this.resources.filter((r) => !this.priceHidden().has(r)),
  );

  private readonly priceValues = computed(() =>
    this.priceVisible().map((resource) => {
      const i = RESOURCE_ORDER.indexOf(resource);
      return {
        resource,
        values: priceSeries(this.priceLog(), i, this.priceMode(), this.basePrices()),
      };
    }),
  );

  /** Index of the last logged day at or before the window's left edge (its price is still in force). */
  private readonly priceStart = computed(() => {
    const log = this.priceLog();
    let start = 0;
    log.forEach((p, i) => {
      if (p.day <= this.xMin()) {
        start = i;
      }
    });
    return start;
  });

  private readonly priceTicks = computed(() => {
    const start = this.priceStart();
    const max = Math.max(1, ...this.priceValues().flatMap((s) => s.values.slice(start)));
    return niceTicks(0, max, 3);
  });

  private priceY(v: number): number {
    const t = this.priceTicks();
    return (
      TOP +
      ((t[t.length - 1] - v) / (t[t.length - 1] - t[0])) * (this.stockPlotHeight() - TOP - BOTTOM)
    );
  }

  protected readonly priceGrid = computed(() => {
    const percent = this.priceMode() === 'percent';
    return this.priceTicks().map((v) => ({
      y: this.priceY(v),
      label: percent ? `${v}%` : formatCompactMoney(v),
    }));
  });

  protected readonly priceEnough = computed(() => this.priceLog().length >= 2);

  protected readonly priceLines = computed(() => {
    const log = this.priceLog();
    if (log.length === 0) {
      return [];
    }
    // The last price holds until the axis ends, so extend the line to the right edge.
    const xs = [...log.map((p) => this.xPx(p.day)), this.xPx(this.xMax())];
    return this.priceValues().map(({ resource, values }) => ({
      resource,
      color: `var(--series-${resource})`,
      d: stepPath(
        xs,
        [...values, values[values.length - 1]].map((v) => this.priceY(v)),
      ),
    }));
  });

  /** Shaded spans for days with a market event, drawn behind all three charts. */
  protected readonly priceBands = computed(() =>
    eventBands(this.priceLog(), this.xMax())
      .filter((b) => b.to > this.xMin())
      .map((b) => ({
        x: this.xPx(Math.max(b.from, this.xMin())),
        width: Math.max(0, this.xPx(b.to) - this.xPx(Math.max(b.from, this.xMin()))),
        events: b.events.join(', '),
      })),
  );

  /** The day's prices under the crosshair, formatted for the current mode. */
  protected readonly hoverPrices = computed(() => {
    const h = this.hoverPoint();
    const day = h ? priceAt(this.priceLog(), h.x) : null;
    if (!day) {
      return null;
    }
    const percent = this.priceMode() === 'percent';
    return {
      heading: `Day ${day.day}${day.events.length ? `: ${day.events.join(', ')}` : ''}`,
      rows: this.resources.map((resource, i) => ({
        resource,
        text: percent
          ? `${priceSeries([day], i, 'percent', this.basePrices())[0]}%`
          : formatMoney(day.prices[i] ?? 0),
      })),
    };
  });

  protected readonly priceSummary = computed(() =>
    this.priceMode() === 'percent'
      ? 'Price of each resource over time, as a percent of its base price. Use the table view for exact values.'
      : 'Price of each resource over time, in dollars. Use the table view for exact values.',
  );

  protected togglePrice(resource: Resource): void {
    const next = new Set(this.priceHidden());
    if (next.has(resource)) {
      next.delete(resource);
    } else if (this.priceVisible().length > 1) {
      next.add(resource);
    }
    this.priceHidden.set(next);
  }

  // ---- hover & keyboard -----------------------------------------------------------

  protected readonly hoverPoint = computed(() => {
    const i = this.hoverIndex();
    return i === null ? null : (this.placed()[i] ?? null);
  });

  /** Market events active on the day under the crosshair, for the capital and stock tooltips. */
  protected readonly hoverEvents = computed(() => {
    const h = this.hoverPoint();
    return (h ? priceAt(this.priceLog(), h.x)?.events : null) ?? [];
  });

  protected readonly hoverX = computed(() => {
    const p = this.hoverPoint();
    return p ? this.xPx(p.x) : null;
  });

  /** Keeps the tooltip inside the chart by anchoring it to the nearer edge. */
  protected readonly tipSide = computed(() =>
    (this.hoverX() ?? 0) > this.width() / 2 ? 'left' : 'right',
  );

  protected onMove(event: PointerEvent, chart: 'capital' | 'stock' | 'price'): void {
    const box = (event.currentTarget as Element).getBoundingClientRect();
    const px = (event.clientX - box.left) * (this.width() / (box.width || this.width()));
    // Points scrolled out of the range can't be hovered: push them out of reach.
    const shown = this.firstShown();
    const i = nearestIndex(
      this.xs().map((x, idx) => (idx < shown ? -1e9 : x)),
      px,
    );
    if (i >= 0) {
      this.hoverIndex.set(i);
      this.hoverChart.set(chart);
    }
  }

  /** A lifted finger leaves the readout up, so a tap can be read; a mouse leaving clears it. */
  protected onLeave(event: PointerEvent): void {
    if (event.pointerType !== 'touch') {
      this.clearHover();
    }
  }

  protected clearHover(): void {
    this.hoverIndex.set(null);
    this.hoverChart.set(null);
  }

  protected onKey(event: KeyboardEvent): void {
    const n = this.placed().length;
    if (n === 0) {
      return;
    }
    const current = this.hoverIndex();
    let next: number | null = null;
    switch (event.key) {
      case 'ArrowLeft':
        next = Math.max(this.firstShown(), (current ?? n) - 1);
        break;
      case 'ArrowRight':
        next = Math.min(n - 1, (current ?? -1) + 1);
        break;
      case 'Home':
        next = this.firstShown();
        break;
      case 'End':
        next = n - 1;
        break;
      case 'Escape':
        this.clearHover();
        event.preventDefault();
        return;
      default:
        return;
    }
    event.preventDefault();
    this.hoverIndex.set(next);
    this.hoverChart.set(this.hoverChart() ?? 'capital');
  }

  // ---- time range & table view ------------------------------------------------------

  protected setRange(days: number | null): void {
    this.range.set(days);
    this.clearHover(); // the point under the cursor may have scrolled out of range
  }

  /** The table lists what the charts show: points and days inside the range. */
  protected readonly tablePoints = computed(() =>
    this.points().filter((_, i) => i >= this.firstShown()),
  );
  protected readonly tablePrices = computed(() => this.priceLog().slice(this.priceStart()));

  // ---- legend ---------------------------------------------------------------------

  protected toggle(resource: Resource): void {
    const next = new Set(this.hidden());
    if (next.has(resource)) {
      next.delete(resource);
    } else if (this.visible().length > 1) {
      next.add(resource); // the last visible line can't be hidden
    }
    this.hidden.set(next);
  }
}
