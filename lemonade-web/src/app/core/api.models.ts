// API contract between lemonade-web and lemonade-api. The backend must return exactly
// these shapes. All money is whole dollars (integers); quantities are cases.

export type Resource = 'lemon' | 'sugar' | 'ice' | 'cup' | 'lemonade';
export type FacilityType = 'warehouse' | 'production';
export type GameStatus = 'active' | 'bankrupt';

export interface User {
  id: number;
  username: string;
}

/** One market-and-inventory row. `price` is the effective price (walk × events). */
export interface ResourceView {
  resource: Resource;
  stock: number;
  capacity: number;
  price: number;
  /** Yesterday's effective price, for the trend arrow; null on day 1. */
  previousPrice: number | null;
  bid: number;
  ask: number;
  /** Effective prices, oldest first, at most 14. */
  history: number[];
}

export interface UpgradeOption {
  tierName: string;
  costPerBuilding: number;
  /** costPerBuilding × buildings. */
  totalCost: number;
  /** Size (cases) or rate (lemonade/day) per building after the upgrade. */
  sizePerBuilding: number;
}

/** Fields shared by both facility types. Level, tier and upgrade apply to the whole type. */
export interface FacilityTypeView {
  tierName: string;
  level: number;
  maxLevel: number;
  /** Total buildings of this type (all five warehouses, or all Production buildings). */
  buildings: number;
  /** Max buildings per warehouse resource, or for Production. */
  maxCount: number;
  /** Cases per warehouse building, or lemonade per day per Production building. */
  sizePerBuilding: number;
  /** Cost to add one building at the current level. */
  expandCost: number;
  /** Total upkeep for this type: buildings × upkeep(level). */
  upkeepPerDay: number;
  /** Null at max level. */
  upgrade: UpgradeOption | null;
}

export interface WarehouseResourceView {
  resource: Resource;
  count: number;
  capacity: number;
}

export interface WarehouseView extends FacilityTypeView {
  resources: WarehouseResourceView[];
}

export interface ProductionView extends FacilityTypeView {
  /** Lemonade per day: buildings × sizePerBuilding. */
  ratePerDay: number;
}

export interface GameEvent {
  key: string;
  name: string;
  description: string;
  multipliers: Partial<Record<Resource, number>>;
  daysLeft: number;
}

export interface GameView {
  day: number;
  capital: number;
  status: GameStatus;
  /** Total upkeep across both facility types. */
  upkeepPerDay: number;
  /** In display order: lemon, sugar, ice, cup, lemonade. */
  resources: ResourceView[];
  facilities: {
    warehouse: WarehouseView;
    production: ProductionView;
  };
  events: GameEvent[];
}

export interface PriceChange {
  resource: Resource;
  before: number;
  after: number;
}

/** Summary of one end-of-day transition. `day` is the day that just ended. */
export interface DayReport {
  day: number;
  produced: number;
  iceMelted: number;
  upkeepPaid: number;
  capitalBefore: number;
  capitalAfter: number;
  priceChanges: PriceChange[];
  newEvents: GameEvent[];
  expiredEvents: GameEvent[];
  bankrupt: boolean;
}

export interface EndDayResponse {
  report: DayReport;
  game: GameView;
}

/** Error body for every non-2xx response. */
export interface ApiError {
  error: string;
  message: string;
}
