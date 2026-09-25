import {
  DayReport,
  GameStats,
  GameView,
  RunDetail,
  RunSummary,
  ScoreRow,
  TradeAmountKey,
  TradeLadder,
  TradeQuote,
  Resource,
  ResourceView,
  SaleInfo,
  TimelinePoint,
} from '../api.models';

const AMOUNTS: [TradeAmountKey, number][] = [
  ['1', 1],
  ['10', 10],
  ['50', 50],
  ['100', 100],
  ['all', 1_000_000],
];

/** The trade ladder for a market with no price impact, as the server would price it. */
export function tradeLadder(
  ask: number,
  bid: number,
  stock: number,
  capacity: number,
  capital = 1000,
): TradeLadder {
  const quote = (qty: number, unit: number): TradeQuote => ({
    qty,
    total: qty * unit,
    averagePrice: qty > 0 ? unit : 0,
    slippagePercent: 0,
  });
  const ladder = { buy: {}, sell: {} } as unknown as TradeLadder;
  for (const [key, amount] of AMOUNTS) {
    const buyQty = Math.max(0, Math.min(amount, Math.floor(capital / ask), capacity - stock));
    ladder.buy[key] = quote(buyQty, ask);
    ladder.sell[key] = quote(Math.min(amount, stock), bid);
  }
  return ladder;
}

function row(resource: Resource, price: number, stock = 0): ResourceView {
  const bid = Math.max(1, Math.floor(price * 0.9));
  const ask = Math.ceil(price * 1.1);
  return {
    resource,
    stock,
    capacity: 10,
    price,
    previousPrice: null,
    bid,
    ask,
    buyDepthLeft: 80,
    sellDepthLeft: 80,
    buyImpactPercent: 0,
    sellImpactPercent: 0,
    trade: tradeLadder(ask, bid, stock, 10),
    history: [price],
    avgCost: 0,
    unrealizedGain: 0,
  };
}

/** A fresh game as the API returns it: day 1, $1,000, everything at level 1. */
export function newGameView(overrides: Partial<GameView> = {}): GameView {
  return {
    day: 1,
    capital: 1000,
    status: 'active',
    upkeepPerDay: 15,
    resources: [
      row('lemon', 20),
      row('sugar', 10),
      row('ice', 10),
      row('cup', 10),
      row('lemonade', 100),
    ],
    facilities: {
      warehouse: {
        tierName: 'Pantry',
        level: 1,
        maxLevel: 4,
        buildings: 5,
        maxCount: 10,
        sizePerBuilding: 10,
        expandCost: 100,
        upkeepPerDay: 5,
        upgrade: {
          tierName: 'Garage',
          costPerBuilding: 155,
          totalCost: 775,
          sizePerBuilding: 25,
          upkeepIncrease: 10,
        },
        marketDepth: 80,
        upgradeMarketDepth: 280,
        resources: (['lemon', 'sugar', 'ice', 'cup', 'lemonade'] as Resource[]).map((resource) => ({
          resource,
          count: 1,
          capacity: 10,
          ...noSale(50),
        })),
      },
      production: {
        tierName: 'Kitchen',
        level: 1,
        maxLevel: 4,
        buildings: 1,
        maxCount: 10,
        sizePerBuilding: 10,
        expandCost: 500,
        upkeepPerDay: 10,
        ratePerDay: 10,
        ...noSale(250),
        upgrade: {
          tierName: 'Food Truck',
          costPerBuilding: 1000,
          totalCost: 1000,
          sizePerBuilding: 20,
          upkeepIncrease: 15,
        },
      },
    },
    events: [],
    timeline: [timelinePoint()],
    stats: gameStats(),
    projection: { lemonadeToProduce: 0, iceToMelt: 0, limitedBy: 'lemon' },
    priceLog: [{ day: 1, prices: [20, 10, 10, 10, 90], events: [] }],
    basePrices: [20, 10, 10, 10, 90],
    netWorth: { cash: 1000, stock: 0, facilities: 500, total: 1500 },
    runId: 'run-current',
    best: null,
    ...overrides,
  };
}

export function dayReport(overrides: Partial<DayReport> = {}): DayReport {
  return {
    day: 4,
    produced: 10,
    iceMelted: 2,
    upkeepPaid: 15,
    forcedSaleCases: 0,
    forcedSaleProceeds: 0,
    capitalBefore: 1240,
    capitalAfter: 1225,
    priceChanges: [
      { resource: 'lemon', before: 20, after: 20 },
      { resource: 'sugar', before: 10, after: 9 },
      { resource: 'ice', before: 10, after: 10 },
      { resource: 'cup', before: 10, after: 10 },
      { resource: 'lemonade', before: 100, after: 140 },
    ],
    newEvents: [],
    expiredEvents: [],
    bankrupt: false,
    ...overrides,
  };
}

/** One timeline point; defaults to the start of a new game. */
export function timelinePoint(overrides: Partial<TimelinePoint> = {}): TimelinePoint {
  return {
    day: 1,
    kind: 'start',
    qty: 0,
    amount: 0,
    produced: 0,
    capital: 1000,
    stock: [0, 0, 0, 0, 0],
    ...overrides,
  };
}

export function gameStats(overrides: Partial<GameStats> = {}): GameStats {
  return {
    casesBought: 0,
    casesSold: 0,
    spent: 0,
    earned: 0,
    facilitiesBought: 0,
    upgrades: 0,
    facilitySpend: 0,
    facilitiesSold: 0,
    facilityProceeds: 0,
    produced: 0,
    upkeepPaid: 0,
    peakCapital: 1000,
    peakDay: 1,
    ...overrides,
  };
}

/** Sale info for a lone building: it is the last one, so it cannot be sold. */
export function noSale(sellValue: number): SaleInfo {
  return { sellValue, canSell: false, sellBlockedReason: 'min_facility', casesToSell: 0 };
}

export function scoreRow(overrides: Partial<ScoreRow> = {}): ScoreRow {
  return {
    rank: 1,
    username: 'lemonjoe',
    score: 2500,
    days: 14,
    netWorth: 2500,
    createdAt: '2026-09-20T12:00:00Z',
    isMe: false,
    ...overrides,
  };
}

export function runSummary(overrides: Partial<RunSummary> = {}): RunSummary {
  return {
    runId: 'run-1',
    score: 1500,
    days: 6,
    netWorth: 1500,
    capital: 900,
    endedBy: 'gave_up',
    createdAt: '2026-09-20T12:00:00Z',
    isBest: false,
    ...overrides,
  };
}

export function runDetail(overrides: Partial<RunDetail> = {}): RunDetail {
  return {
    ...runSummary(),
    stats: gameStats(),
    timeline: [timelinePoint(), timelinePoint({ kind: 'end_day', day: 1 })],
    priceLog: [
      { day: 1, prices: [20, 10, 10, 10, 90], events: [] },
      { day: 2, prices: [22, 9, 10, 11, 100], events: ['Heat Wave'] },
    ],
    basePrices: [20, 10, 10, 10, 90],
    reports: [],
    ...overrides,
  };
}
