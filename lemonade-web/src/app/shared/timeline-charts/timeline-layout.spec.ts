import { timelinePoint } from '../../core/testing/fixtures';
import {
  describePoint,
  formatCompactMoney,
  dayTicks,
  markerPoints,
  nearestIndex,
  niceTicks,
  placePoints,
  stepPath,
} from './timeline-layout';

describe('placePoints', () => {
  it('puts the start at the beginning of day 1 and end of day at the next day boundary', () => {
    const placed = placePoints([
      timelinePoint({ kind: 'start' }),
      timelinePoint({ kind: 'end_day', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 2 }),
    ]);
    expect(placed.map((p) => p.x)).toEqual([1, 2, 3]);
  });

  it('spreads a days actions evenly between its boundaries', () => {
    const placed = placePoints([
      timelinePoint({ kind: 'start' }),
      timelinePoint({ kind: 'buy', day: 1 }),
      timelinePoint({ kind: 'sell', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 1 }),
      timelinePoint({ kind: 'buy', day: 2 }),
    ]);
    const xs = placed.map((p) => p.x);
    expect(xs[1]).toBeCloseTo(1 + 1 / 3);
    expect(xs[2]).toBeCloseTo(1 + 2 / 3);
    expect(xs[3]).toBe(2);
    expect(xs[4]).toBeCloseTo(2.5);
  });

  it('keeps points in order and remembers their original index', () => {
    const placed = placePoints([
      timelinePoint({ kind: 'start' }),
      timelinePoint({ kind: 'buy', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 1 }),
    ]);
    expect(placed.map((p) => p.index)).toEqual([0, 1, 2]);
    expect(placed.map((p) => p.x)).toEqual([...placed.map((p) => p.x)].sort((a, b) => a - b));
  });

  it('copes with days that have no recorded actions (old days are compacted)', () => {
    const placed = placePoints([
      timelinePoint({ kind: 'start' }),
      timelinePoint({ kind: 'end_day', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 2 }),
      timelinePoint({ kind: 'end_day', day: 3 }),
      timelinePoint({ kind: 'buy', day: 4 }),
    ]);
    expect(placed.map((p) => p.x)).toEqual([1, 2, 3, 4, 4.5]);
  });
});

describe('niceTicks', () => {
  it('returns round numbers that cover the range', () => {
    expect(niceTicks(0, 1234, 4)).toEqual([0, 500, 1000, 1500]);
    expect(niceTicks(0, 10, 5)).toEqual([0, 2, 4, 6, 8, 10]);
  });

  it('handles a flat or empty range', () => {
    const ticks = niceTicks(0, 0, 4);
    expect(ticks[0]).toBe(0);
    expect(ticks[ticks.length - 1]).toBeGreaterThan(0);
  });

  it('includes negative and large values', () => {
    const ticks = niceTicks(-50, 480000, 4);
    expect(ticks[0]).toBeLessThanOrEqual(-50);
    expect(ticks[ticks.length - 1]).toBeGreaterThanOrEqual(480000);
  });
});

describe('dayTicks', () => {
  it('labels every day when there is room', () => {
    expect(dayTicks(1, 5, 10)).toEqual([1, 2, 3, 4, 5]);
  });

  it('thins the labels on long games', () => {
    const ticks = dayTicks(1, 100, 8);
    expect(ticks.length).toBeLessThanOrEqual(8);
    expect(ticks[0]).toBe(1);
    expect(ticks.every((t) => Number.isInteger(t))).toBeTrue();
  });
});

describe('stepPath', () => {
  it('holds each value until the next action, then jumps', () => {
    expect(stepPath([0, 10, 20], [5, 15, 10])).toBe('M0 5H10V15H20V10');
  });

  it('is empty for no points', () => {
    expect(stepPath([], [])).toBe('');
  });
});

describe('nearestIndex', () => {
  it('finds the closest x', () => {
    expect(nearestIndex([0, 10, 20, 100], 4)).toBe(0);
    expect(nearestIndex([0, 10, 20, 100], 16)).toBe(2);
    expect(nearestIndex([0, 10, 20, 100], 90)).toBe(3);
  });

  it('is -1 when there are no points', () => {
    expect(nearestIndex([], 5)).toBe(-1);
  });
});

describe('markerPoints', () => {
  it('keeps only buys, sells, and facility purchases', () => {
    const placed = placePoints([
      timelinePoint({ kind: 'start' }),
      timelinePoint({ kind: 'buy', day: 1 }),
      timelinePoint({ kind: 'sell', day: 1 }),
      timelinePoint({ kind: 'expand', day: 1 }),
      timelinePoint({ kind: 'upgrade', day: 1 }),
      timelinePoint({ kind: 'facility_sold', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 1 }),
    ]);
    expect(markerPoints(placed).map((p) => p.point.kind)).toEqual([
      'buy',
      'sell',
      'expand',
      'upgrade',
      'facility_sold',
    ]);
  });
});

describe('describePoint', () => {
  it('describes each kind of point in plain words', () => {
    expect(describePoint(timelinePoint())).toBe('Game started');
    expect(
      describePoint(timelinePoint({ kind: 'buy', resource: 'lemon', qty: 3, amount: 66 })),
    ).toBe('Bought 3 lemon for $66');
    expect(
      describePoint(timelinePoint({ kind: 'sell', resource: 'lemonade', qty: 2, amount: 162 })),
    ).toBe('Sold 2 lemonade for $162');
    expect(
      describePoint(
        timelinePoint({ kind: 'expand', facility: 'warehouse', resource: 'ice', amount: 100 }),
      ),
    ).toBe('Built an ice warehouse for $100');
    expect(
      describePoint(timelinePoint({ kind: 'expand', facility: 'production', amount: 500 })),
    ).toBe('Built a production building for $500');
    expect(
      describePoint(timelinePoint({ kind: 'upgrade', facility: 'warehouse', amount: 500 })),
    ).toBe('Upgraded all warehouses for $500');
    expect(
      describePoint(timelinePoint({ kind: 'upgrade', facility: 'production', amount: 1000 })),
    ).toBe('Upgraded production for $1,000');
    expect(
      describePoint(
        timelinePoint({
          kind: 'facility_sold',
          facility: 'warehouse',
          resource: 'ice',
          amount: 50,
        }),
      ),
    ).toBe('Sold an ice warehouse for $50');
    expect(
      describePoint(timelinePoint({ kind: 'facility_sold', facility: 'production', amount: 250 })),
    ).toBe('Sold a production building for $250');
    expect(
      describePoint(timelinePoint({ kind: 'end_day', day: 4, produced: 10, amount: 30 })),
    ).toBe('Day 4 ended: made 10 lemonade, paid $30 upkeep');
  });
});

describe('formatCompactMoney', () => {
  it('shortens big numbers for axis labels', () => {
    expect(formatCompactMoney(0)).toBe('$0');
    expect(formatCompactMoney(950)).toBe('$950');
    expect(formatCompactMoney(1500)).toBe('$1.5k');
    expect(formatCompactMoney(12000)).toBe('$12k');
    expect(formatCompactMoney(2_500_000)).toBe('$2.5M');
    expect(formatCompactMoney(-500)).toBe('-$500');
  });
});
