import { PricePoint, Resource, TimelinePoint } from '../../core/api.models';
import { RESOURCE_LABELS } from '../../core/resources';
import { formatMoney } from '../money.pipe';

// Pure layout and wording helpers for the two history charts. Nothing here touches
// the DOM, so it is all unit-tested directly.

export interface PlotPoint {
  point: TimelinePoint;
  /** Game time: day d spans [d, d + 1). Actions fall inside their day; end of day sits on the boundary. */
  x: number;
  /** Position in the original timeline array. */
  index: number;
}

/**
 * Assigns each point a position on the shared time axis. A day's actions are spread
 * evenly across the day (the API only says which day, not when), the start sits at
 * the beginning of its day, and end-of-day points sit on the boundary that follows.
 */
export function placePoints(points: TimelinePoint[]): PlotPoint[] {
  const actionsPerDay = new Map<number, number>();
  for (const p of points) {
    if (p.kind !== 'start' && p.kind !== 'end_day') {
      actionsPerDay.set(p.day, (actionsPerDay.get(p.day) ?? 0) + 1);
    }
  }
  const seen = new Map<number, number>();
  return points.map((point, index) => {
    let x: number;
    if (point.kind === 'start') {
      x = point.day;
    } else if (point.kind === 'end_day') {
      x = point.day + 1;
    } else {
      const k = (seen.get(point.day) ?? 0) + 1;
      seen.set(point.day, k);
      x = point.day + k / ((actionsPerDay.get(point.day) ?? 1) + 1);
    }
    return { point, x, index };
  });
}

/** Round tick values that cover [min, max], e.g. 0, 500, 1000, 1500. */
export function niceTicks(min: number, max: number, count: number): number[] {
  if (max <= min) {
    max = min + 1;
  }
  const rough = (max - min) / count;
  const pow = Math.pow(10, Math.floor(Math.log10(rough)));
  const fraction = rough / pow;
  const step = (fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 5 ? 5 : 10) * pow;
  const start = Math.floor(min / step) * step;
  const end = Math.ceil(max / step) * step;
  const ticks: number[] = [];
  for (let i = 0; start + i * step <= end + step / 1e6; i++) {
    ticks.push(Math.round((start + i * step) * 1e6) / 1e6);
  }
  return ticks;
}

/** Whole-day labels for the time axis, thinned to at most maxTicks on long games. */
export function dayTicks(first: number, last: number, maxTicks: number): number[] {
  const span = last - first + 1;
  let step = 1;
  for (const candidate of [1, 2, 5, 10, 20, 50, 100, 200, 500, 1000]) {
    step = candidate;
    if (span / candidate <= maxTicks) {
      break;
    }
  }
  const ticks = [first];
  for (let d = Math.ceil((first + 1) / step) * step; d <= last; d += step) {
    ticks.push(d);
  }
  return ticks;
}

const round = (n: number) => Math.round(n * 100) / 100;

/**
 * SVG path for a value that holds until the next action and then jumps, which is how
 * capital and stock actually behave: flat between actions, a step when one happens.
 */
export function stepPath(xs: number[], ys: number[]): string {
  if (xs.length === 0) {
    return '';
  }
  let d = `M${round(xs[0])} ${round(ys[0])}`;
  for (let i = 1; i < xs.length; i++) {
    d += `H${round(xs[i])}V${round(ys[i])}`;
  }
  return d;
}

/** Index of the x closest to px (xs ascending), or -1 when empty. */
export function nearestIndex(xs: number[], px: number): number {
  let best = -1;
  let bestDistance = Infinity;
  for (let i = 0; i < xs.length; i++) {
    const distance = Math.abs(xs[i] - px);
    if (distance < bestDistance) {
      best = i;
      bestDistance = distance;
    }
  }
  return best;
}

/** The points worth a dot on the capital chart: trades and facility purchases. */
export function markerPoints(placed: PlotPoint[]): PlotPoint[] {
  return placed.filter((p) =>
    ['buy', 'sell', 'expand', 'upgrade', 'facility_sold'].includes(p.point.kind),
  );
}

const NOUN: Record<Resource, string> = {
  lemon: 'lemon',
  sugar: 'sugar',
  ice: 'ice',
  cup: 'cup',
  lemonade: 'lemonade',
};

const article = (word: string) => (/^[aeiou]/.test(word) ? 'an' : 'a');

/** One plain-language line per point, used by the tooltip and the table view. */
export function describePoint(p: TimelinePoint): string {
  const resource = p.resource ? RESOURCE_LABELS[p.resource].toLowerCase() : '';
  switch (p.kind) {
    case 'start':
      return 'Game started';
    case 'buy':
      return `Bought ${p.qty} ${resource} for ${formatMoney(p.amount)}`;
    case 'sell':
      return `Sold ${p.qty} ${resource} for ${formatMoney(p.amount)}`;
    case 'expand':
      if (p.facility === 'warehouse' && p.resource) {
        const noun = NOUN[p.resource];
        return `Built ${article(noun)} ${noun} warehouse for ${formatMoney(p.amount)}`;
      }
      return `Built a production building for ${formatMoney(p.amount)}`;
    case 'upgrade':
      return p.facility === 'warehouse'
        ? `Upgraded all warehouses for ${formatMoney(p.amount)}`
        : `Upgraded production for ${formatMoney(p.amount)}`;
    case 'facility_sold':
      if (p.facility === 'warehouse' && p.resource) {
        const noun = NOUN[p.resource];
        return `Sold ${article(noun)} ${noun} warehouse for ${formatMoney(p.amount)}`;
      }
      return `Sold a production building for ${formatMoney(p.amount)}`;
    case 'end_day':
      return `Day ${p.day} ended: made ${p.produced} lemonade, paid ${formatMoney(p.amount)} upkeep`;
  }
}

/** $950, $1.5k, $12k, $2.5M: short enough for an axis. */
export function formatCompactMoney(n: number): string {
  const abs = Math.abs(n);
  const trim = (v: number) => String(Math.round(v * 10) / 10);
  const body =
    abs >= 1e6 ? `${trim(abs / 1e6)}M` : abs >= 1000 ? `${trim(abs / 1000)}k` : String(abs);
  return `${n < 0 ? '-' : ''}$${body}`;
}

export type PriceMode = 'dollars' | 'percent';

/** One resource's line for the price chart: dollars, or percent of its base price. */
export function priceSeries(
  log: PricePoint[],
  index: number,
  mode: PriceMode,
  basePrices: number[],
): number[] {
  return log.map((p) => {
    const price = p.prices[index] ?? 0;
    const base = basePrices[index];
    return mode === 'percent' && base ? Math.round((price / base) * 1000) / 10 : price;
  });
}

/** The prices in force at game time x: the newest log point that has started by then. */
export function priceAt(log: PricePoint[], x: number): PricePoint | null {
  let found: PricePoint | null = null;
  for (const p of log) {
    if (p.day <= x) {
      found = p;
    }
  }
  return found;
}

export interface EventBand {
  from: number;
  to: number;
  events: string[];
}

/** Spans of game time (day d is [d, d + 1)) on which any event was active, clipped to xMax. */
export function eventBands(log: PricePoint[], xMax: number): EventBand[] {
  return log
    .filter((p) => p.events.length > 0 && p.day < xMax)
    .map((p) => ({ from: p.day, to: Math.min(p.day + 1, xMax), events: p.events }));
}
