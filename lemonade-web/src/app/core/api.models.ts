// API contract between lemonade-web and lemonade-api. The backend must return exactly
// these shapes. All money is whole dollars (integers); quantities are cases.

/** A commodity key from the server's catalog (see `Commodity`). */
export type Resource = string;

/** One catalog entry. Per-commodity arrays (timeline stock, price log prices, base prices) follow the catalog order. */
export interface Commodity {
  key: Resource;
  name: string;
  category: string;
  storageClass: string;
  isProduct: boolean;
  order: number;
}
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
  /** `bid` and `ask` are the price of the next single case, including the market's reaction to the player's own recent trades. */
  /** Cases that can still be bought (or sold) at the plain price before the market reacts. */
  buyDepthLeft: number;
  sellDepthLeft: number;
  /** How far the next case is from the plain price, in percent (0 when the market has not reacted). */
  buyImpactPercent: number;
  sellImpactPercent: number;
  /** What the bulk-bar amounts would cost or raise, with price impact, cut to cash, space and stock. */
  trade: TradeLadder;
  /** Average paid per case held (for lemonade, the cost to make one); 0 when none is held. */
  avgCost: number;
  /** Stock value at the bid minus what it cost; negative is a loss. 0 when none is held. */
  unrealizedGain: number;
  /** The 7-day average price in whole dollars; null without a market analyst. */
  movingAverage: number | null;
}

/** What a trade of some size costs (buy) or raises (sell), as the server prices it. */
export interface TradeQuote {
  /** Cases the quote covers: fewer than asked when cash, space or stock run out. */
  qty: number;
  total: number;
  /** Per case, with impact, to one decimal. */
  averagePrice: number;
  /** How much worse than the plain price the trade is, in percent. */
  slippagePercent: number;
}

export type TradeAmountKey = '1' | '10' | '50' | '100' | 'all';
export type TradeSideQuotes = Record<TradeAmountKey, TradeQuote>;

export interface TradeLadder {
  buy: TradeSideQuotes;
  sell: TradeSideQuotes;
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
  /** Cases the market takes at the plain price at this level. */
  marketDepth: number;
  /** The same after the upgrade; 0 at max level. */
  upgradeMarketDepth: number;
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

/** The player's top finished run so far. */
export interface Best {
  score: number;
  days: number;
  runId: string;
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
  /** The commodity catalog, in display order. */
  commodities: Commodity[];
  netWorth: NetWorth;
  /** Names this playthrough. */
  runId: string;
  /** The player's top finished run, or null before they have finished one. */
  best: Best | null;
  /** Convenience features turned on by upgrades, for example `pnl`, `repeat_trades`, `price_alerts`. */
  features: string[];
  /** Cases of ice the freezers keep overnight (0 without one). */
  iceKeepCases: number;
  /** Events the player's upgrades let them see coming, soonest first. */
  forecast: ForecastEntry[];
  /** Achievements this response's mutation just earned; `[]` on a plain read. */
  unlocked: UnlockedAchievement[];
}

/** An event that will start `daysAhead` days from now (1 is tomorrow). */
export interface ForecastEntry {
  daysAhead: number;
  key: string;
  name: string;
  duration: number;
}

export type AchievementTier = 'bronze' | 'silver' | 'gold';

/** One achievement a mutation just unlocked, for the toast. */
export interface UnlockedAchievement {
  key: string;
  name: string;
  tier: AchievementTier;
}

export interface AchievementCategory {
  key: string;
  name: string;
}

/** One row of the achievements page. A locked hidden row has a null name and description. */
export interface Achievement {
  key: string;
  name: string | null;
  description: string | null;
  category: string;
  tier: AchievementTier;
  hidden: boolean;
  unlocked: boolean;
  unlockedAt: string | null;
  /** The run that unlocked it. */
  runId: string | null;
  /** How far the current game is, for measurable goals; null otherwise or once unlocked. */
  progress: { current: number; target: number } | null;
}

export interface AchievementsResponse {
  categories: AchievementCategory[];
  achievements: Achievement[];
  unlockedCount: number;
  total: number;
}

export type EndedBy = 'bankrupt' | 'gave_up';

/** One line of the global board: a player's best run. Other players' runs are not viewable. */
export interface ScoreRow {
  rank: number;
  username: string;
  score: number;
  days: number;
  netWorth: number;
  createdAt: string;
  /** The caller's own row. */
  isMe: boolean;
  /** How many achievements the player has unlocked. */
  achievements: number;
}

/** all_time ranks each player's best finished run; day_100 their net worth on reaching day 100. */
export type Board = 'all_time' | 'day_100';

export interface ScoresResponse {
  /** The board these rows belong to. */
  board: Board;
  rows: ScoreRow[];
  /** The caller's own best row and rank, even below the rows shown; null with no finished run. */
  me: ScoreRow | null;
}

/** One line of the caller's record history. */
export interface RunSummary {
  runId: string;
  score: number;
  days: number;
  netWorth: number;
  capital: number;
  endedBy: EndedBy;
  createdAt: string;
  /** The caller's top run (an earlier finish wins a tie). */
  isBest: boolean;
}

/** A finished run in full; readable only by its owner. */
export interface RunDetail extends RunSummary {
  stats: GameStats;
  timeline: TimelinePoint[];
  priceLog: PricePoint[];
  basePrices: number[];
  commodities: Commodity[];
  /** The run's ended days, as the past-days list returns them. */
  reports: ReportSummary[];
  /** The achievements this run unlocked. */
  achievements: UnlockedAchievement[];
}

/** `limitedBy` is a resource, `production`, `space`, or empty when production capacity is 0. */
export interface Projection {
  lemonadeToProduce: number;
  iceToMelt: number;
  /** Ice a freezer will keep for tomorrow (part of what would otherwise melt). */
  iceKept: number;
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

/** One row of the past-days list; the full report is fetched per day. */
export interface ReportSummary {
  day: number;
  produced: number;
  capitalBefore: number;
  capitalAfter: number;
  /** Event names. */
  newEvents: string[];
  expiredEvents: string[];
  bankrupt: boolean;
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
  /** Ice the freezer kept for the next day. */
  iceKept: number;
  upkeepPaid: number;
  /** The part of upkeep owed for upgrades. */
  upgradeUpkeep: number;
  /** Ice made overnight by an ice machine, and what it cost. */
  iceMade: number;
  iceMadeCost: number;
  /** The bookkeeper's profit and loss for the day; null without a bookkeeper. */
  pnl: Pnl | null;
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

export interface Pnl {
  sales: number;
  purchases: number;
  facilities: number;
  upkeep: number;
  iceMade: number;
  net: number;
}

export type UpgradeState = 'owned' | 'available' | 'locked';
export type UpgradeLockCode = 'era' | 'warehouse_level' | 'production_level' | 'requires_upgrade';

/** One upgrade on the Upgrades page. `lockedReason` is plain words, set only when locked. */
export interface UpgradeItem {
  key: string;
  name: string;
  category: string;
  era: number;
  cost: number;
  upkeep: number;
  text: string;
  state: UpgradeState;
  lockCode: UpgradeLockCode | '';
  lockedReason: string;
}

export interface UpgradeCategory {
  key: string;
  name: string;
}

/** GET /api/game/upgrades. */
export interface UpgradesResponse {
  era: number;
  categories: UpgradeCategory[];
  upgrades: UpgradeItem[];
  ownedCount: number;
  spent: number;
  upkeepPerDay: number;
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
