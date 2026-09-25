import {
  DayReport,
  GameStats,
  GameView,
  Resource,
  ResourceView,
  TimelinePoint,
} from '../api.models';

function row(resource: Resource, price: number, stock = 0): ResourceView {
  return {
    resource,
    stock,
    capacity: 10,
    price,
    previousPrice: null,
    bid: Math.max(1, Math.floor(price * 0.9)),
    ask: Math.ceil(price * 1.1),
    history: [price],
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
          costPerBuilding: 100,
          totalCost: 500,
          sizePerBuilding: 20,
          upkeepIncrease: 10,
        },
        resources: (['lemon', 'sugar', 'ice', 'cup', 'lemonade'] as Resource[]).map((resource) => ({
          resource,
          count: 1,
          capacity: 10,
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
    produced: 0,
    upkeepPaid: 0,
    peakCapital: 1000,
    peakDay: 1,
    ...overrides,
  };
}
