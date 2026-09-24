import { DayReport, GameView, Resource, ResourceView } from '../api.models';

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
    ...overrides,
  };
}

export function dayReport(overrides: Partial<DayReport> = {}): DayReport {
  return {
    day: 4,
    produced: 10,
    iceMelted: 2,
    upkeepPaid: 15,
    capitalBefore: 1240,
    capitalAfter: 1225,
    priceChanges: [{ resource: 'lemonade', before: 100, after: 140 }],
    newEvents: [],
    expiredEvents: [],
    bankrupt: false,
    ...overrides,
  };
}
