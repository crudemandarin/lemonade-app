import {
  Achievement,
  AchievementsResponse,
  Commodity,
  DayReport,
  GameStats,
  GameView,
  RunDetail,
  RunSummary,
  ScoreRow,
  TradeAmountKey,
  TradeLadder,
  TradeQuote,
  UpgradeItem,
  UpgradesResponse,
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
    reach: 100,
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
    movingAverage: null,
    unlocked: true,
    buyable: true,
    shelfDays: 0,
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
        classes: [
          { class: 'cold', name: 'Cold room', count: 1, members: ['lemon'] },
          { class: 'dry', name: 'Dry store', count: 2, members: ['sugar', 'cup'] },
          { class: 'frozen', name: 'Freezer', count: 1, members: ['ice'] },
          { class: 'finished', name: 'Finished goods', count: 1, members: ['lemonade'] },
        ].map((c) => ({
          ...c,
          members: c.members as Resource[],
          capacity: c.count * 10,
          stock: 0,
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
    recipes: [],
    plan: [{ recipe: 'lemonade', target: 0 }],
    projection: {
      lemonadeToProduce: 0,
      iceToMelt: 0,
      iceKept: 0,
      limitedBy: 'lemon',
      plan: [],
      willSpoil: {},
    },
    priceLog: [{ day: 1, prices: [20, 10, 10, 10, 90], events: [] }],
    basePrices: [20, 10, 10, 10, 90],
    commodities: commodities(),
    netWorth: { cash: 1000, stock: 0, facilities: 500, acquisitions: 0, total: 1500 },
    runId: 'run-current',
    best: null,
    features: [],
    iceKeepCases: 0,
    forecast: [],
    unlocked: [],
    era: 1,
    eraName: 'Neighborhood',
    nextGoal: null,
    ...overrides,
  };
}

export function dayReport(overrides: Partial<DayReport> = {}): DayReport {
  return {
    day: 4,
    produced: 10,
    iceMelted: 2,
    iceKept: 0,
    spoiled: {},
    made: [],
    upkeepPaid: 15,
    upgradeUpkeep: 0,
    iceMade: 0,
    iceMadeCost: 0,
    pnl: null,
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
    achievements: 0,
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
    commodities: commodities(),
    reports: [],
    achievements: [],
    ...overrides,
  };
}

/** The default catalog, as the API reports it. */
export function commodities(): Commodity[] {
  return [
    {
      key: 'lemon',
      name: 'Lemons',
      category: 'ingredient',
      storageClass: 'cold',
      isProduct: false,
      order: 10,
      shelfLifeDays: 0,
    },
    {
      key: 'sugar',
      name: 'Sugar',
      category: 'ingredient',
      storageClass: 'dry',
      isProduct: false,
      order: 20,
      shelfLifeDays: 0,
    },
    {
      key: 'ice',
      name: 'Ice',
      category: 'ingredient',
      storageClass: 'frozen',
      isProduct: false,
      order: 30,
      shelfLifeDays: 0,
    },
    {
      key: 'cup',
      name: 'Cups',
      category: 'ingredient',
      storageClass: 'dry',
      isProduct: false,
      order: 40,
      shelfLifeDays: 0,
    },
    {
      key: 'lemonade',
      name: 'Lemonade',
      category: 'product',
      storageClass: 'finished',
      isProduct: true,
      order: 50,
      shelfLifeDays: 0,
    },
  ];
}

export function upgradeItem(overrides: Partial<UpgradeItem> = {}): UpgradeItem {
  return {
    key: 'order_book',
    name: 'Order book',
    category: 'convenience',
    era: 1,
    cost: 500,
    upkeep: 0,
    text: "A button to repeat yesterday's trades.",
    state: 'available',
    lockCode: '',
    lockedReason: '',
    ...overrides,
  };
}

export function achievement(overrides: Partial<Achievement> = {}): Achievement {
  return {
    key: 'nw_5k',
    name: 'Pocket money',
    description: 'Reach a net worth of $5,000.',
    category: 'wealth',
    tier: 'bronze',
    hidden: false,
    unlocked: false,
    unlockedAt: null,
    runId: null,
    progress: null,
    ...overrides,
  };
}

export function upgradesResponse(overrides: Partial<UpgradesResponse> = {}): UpgradesResponse {
  return {
    era: 1,
    categories: [
      { key: 'freshness', name: 'Freshness and storage' },
      { key: 'convenience', name: 'Convenience' },
    ],
    upgrades: [
      upgradeItem({
        key: 'freezer_1',
        name: 'Chest freezer',
        category: 'freshness',
        cost: 1200,
        upkeep: 5,
        text: 'Keep up to 20 cases of ice one extra night.',
      }),
      upgradeItem({
        key: 'freezer_2',
        name: 'Walk-in freezer',
        category: 'freshness',
        era: 2,
        cost: 8000,
        upkeep: 20,
        text: 'Keep up to 150 cases of ice one extra night.',
        state: 'locked',
        lockCode: 'era',
        lockedReason: 'Reach era 2 first.',
      }),
      upgradeItem(),
    ],
    ownedCount: 0,
    spent: 0,
    upkeepPerDay: 0,
    ...overrides,
  };
}

export function achievementsResponse(
  overrides: Partial<AchievementsResponse> = {},
): AchievementsResponse {
  const achievements = overrides.achievements ?? [achievement()];
  return {
    categories: [
      { key: 'wealth', name: 'Wealth' },
      { key: 'oddities', name: 'Oddities' },
    ],
    achievements,
    unlockedCount: achievements.filter((a) => a.unlocked).length,
    total: achievements.length,
    ...overrides,
  };
}
