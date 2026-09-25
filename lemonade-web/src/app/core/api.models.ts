// API contract between lemonade-web and lemonade-api. The backend must return exactly
// these shapes. All money is whole dollars (integers); quantities are cases.

export type Resource = 'lemon' | 'sugar' | 'ice' | 'cup' | 'lemonade';
export type FacilityType = 'warehouse' | 'production';
export type GameStatus = 'active' | 'bankrupt' | 'gave_up';

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
  /** Average paid per case held (for lemonade, the cost to make one); 0 when none is held. */
  avgCost: number;
  /** Stock value at the bid minus what it cost; negative is a loss. 0 when none is held. */
  unrealizedGain: number;
}

export interface UpgradeOption {
  tierName: string;
  costPerBuilding: number;
  /** costPerBuilding × buildings. */
  totalCost: number;
  /** Size (cases) or rate (lemonade/day) per building after the upgrade. */
  sizePerBuilding: number;
  /** Extra upkeep per day for the whole type after the upgrade. */
  upkeepIncrease: number;
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

/** What selling one building would do; part of every warehouse row and of production. */
export interface SaleInfo {
  /** What one building sells for now (build cost at the current level × the resale rate). */
  sellValue: number;
  canSell: boolean;
  /** Empty when `canSell`. */
  sellBlockedReason: '' | 'min_facility' | 'stock_exceeds_capacity';
  /** Cases to sell first when the reason is `stock_exceeds_capacity`, else 0. */
  casesToSell: number;
}

export interface WarehouseResourceView extends SaleInfo {
  resource: Resource;
  count: number;
  capacity: number;
}

export interface WarehouseView extends FacilityTypeView {
  resources: WarehouseResourceView[];
}

export interface ProductionView extends FacilityTypeView, SaleInfo {
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

export type PointKind =
  'start' | 'buy' | 'sell' | 'expand' | 'upgrade' | 'facility_sold' | 'end_day';

/** The state right after one action, for the history charts. */
export interface TimelinePoint {
  /** The day the action happened on (for end_day, the day that just ended). */
  day: number;
  kind: PointKind;
  /** Buys, sells, and warehouse expansions. */
  resource?: Resource;
  /** Expansions and upgrades. */
  facility?: FacilityType;
  /** Cases bought or sold (repeat clicks are merged); buildings added. */
  qty: number;
  /** Dollars spent (buy, expand, upgrade), earned (sell), or upkeep paid (end_day). */
  amount: number;
  /** Lemonade made overnight (end_day only). */
  produced: number;
  capital: number;
  /** Stock after the action, in order: lemon, sugar, ice, cup, lemonade. */
  stock: number[];
}

/** Running totals for the end-of-game report. */
export interface GameStats {
  casesBought: number;
  casesSold: number;
  spent: number;
  earned: number;
  facilitiesBought: number;
  upgrades: number;
  facilitySpend: number;
  facilitiesSold: number;
  /** Cash from selling buildings. */
  facilityProceeds: number;
  produced: number;
  upkeepPaid: number;
  peakCapital: number;
  peakDay: number;
}

/** What the player is worth today. The score of a finished run is its final `total`. */
export interface NetWorth {
  cash: number;
  /** Stock at the current bid. */
  stock: number;
  /** What every building would resell for. */
  facilities: number;
  total: number;
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
  /** Oldest first. Old days are compacted to milestones only. */
  timeline: TimelinePoint[];
  stats: GameStats;
  /** What End day would do right now; recomputed by the server on every view. */
  projection: Projection;
  /** One point per day, oldest first. Older saves start at the day they were upgraded. */
  priceLog: PricePoint[];
  /** Long-run prices in the same order as `PricePoint.prices`, for the "% of base" view. */
  basePrices: number[];
  netWorth: NetWorth;
}

/** `limitedBy` is a resource, `production`, `space`, or empty when production capacity is 0. */
export interface Projection {
  lemonadeToProduce: number;
  iceToMelt: number;
  limitedBy: Resource | 'production' | 'space' | '';
}

/** One day of the price chart: effective prices as the player saw them that day. */
export interface PricePoint {
  day: number;
  /** In order: lemon, sugar, ice, cup, lemonade. */
  prices: number[];
  /** Names of the events active that day. */
  events: string[];
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
  /** Stock sold at bid because cash alone could not cover upkeep (0 when none). */
  forcedSaleCases: number;
  forcedSaleProceeds: number;
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
