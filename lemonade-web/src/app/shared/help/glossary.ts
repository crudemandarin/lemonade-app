import { Resource } from '../../core/api.models';
import { IconName } from '../icon/icon.component';

export type HelpSection =
  'quickstart' | 'basics' | 'market' | 'resources' | 'facilities' | 'upgrades' | 'events';

export interface HelpTerm {
  id: string;
  section: HelpSection;
  term: string;
  definition: string;
  /** Line icon, or a resource picture when `resource` is set. */
  icon?: IconName;
  resource?: Resource;
}

export const HELP_SECTIONS: { id: HelpSection; label: string; icon: IconName }[] = [
  { id: 'quickstart', label: 'Quick start', icon: 'end-day' },
  { id: 'basics', label: 'Basics', icon: 'calendar' },
  { id: 'market', label: 'Market', icon: 'coin' },
  { id: 'resources', label: 'Resources', icon: 'warehouse' },
  { id: 'facilities', label: 'Facilities', icon: 'factory' },
  { id: 'upgrades', label: 'Upgrades', icon: 'upgrade' },
  { id: 'events', label: 'Events', icon: 'event' },
];

export interface QuickStep {
  icon: IconName;
  title: string;
  text: string;
}

export const QUICK_START: QuickStep[] = [
  {
    icon: 'coin',
    title: 'Buy ingredients',
    text: 'Stock up on lemons, sugar, ice and cups while prices are low.',
  },
  {
    icon: 'factory',
    title: 'Check the projection',
    text: 'The header shows how much lemonade you will make, and what is holding you back.',
  },
  {
    icon: 'end-day',
    title: 'End the day',
    text: 'Lemonade is made, ice melts and upkeep is paid.',
  },
  {
    icon: 'trend-up',
    title: 'Sell high',
    text: 'Sell your lemonade when the price is up. Events can swing it a lot.',
  },
  {
    icon: 'upgrade',
    title: 'Grow',
    text: 'Reinvest in bigger warehouses and production, but keep cash for upkeep.',
  },
];

// Numbers in the prose mirror the backend Config (lemonade-api/internal/domain/config.go):
// Spread (10%), the 50% resale rate and the event table. Update both together.
export const HELP_TERMS: HelpTerm[] = [
  {
    id: 'recipe',
    resource: 'lemonade',
    section: 'basics',
    term: 'The recipe',
    definition:
      'One lemon, one sugar, one ice and one cup make one lemonade. Buy the ingredients low, and sell the lemonade high.',
  },
  {
    id: 'day',
    icon: 'calendar',
    section: 'basics',
    term: 'Days',
    definition:
      'The game is turn based. Nothing changes until you press End day, so take as long as you like.',
  },
  {
    id: 'capital',
    icon: 'coin',
    section: 'basics',
    term: 'Capital',
    definition:
      'The cash you have on hand. You spend it on stock and buildings, and earn it by selling.',
  },
  {
    id: 'net-worth',
    icon: 'trend-up',
    section: 'basics',
    term: 'Net worth',
    definition:
      'Your capital plus the value of your stock, at the price you could sell it for right now.',
  },
  {
    id: 'bankrupt',
    icon: 'sad-face',
    section: 'basics',
    term: 'Going bankrupt',
    definition:
      'Upkeep is due every day. If your cash falls short, stock is sold at the bid to cover it: lemonade first, then lemon, sugar, cups and ice. If that still is not enough, the game is over.',
  },

  {
    id: 'price',
    icon: 'trend-flat',
    section: 'market',
    term: 'Price',
    definition:
      'Prices drift a little each day but are pulled back toward a normal level. Events can push them well above or below it.',
  },
  {
    id: 'bid',
    icon: 'trend-down',
    section: 'market',
    term: 'Bid',
    definition: 'What the market pays you when you sell. It sits 10% below the price.',
  },
  {
    id: 'ask',
    icon: 'trend-up',
    section: 'market',
    term: 'Ask',
    definition:
      'What the market charges you when you buy. It sits 10% above the price, so buying and selling straight away always loses a little.',
  },
  {
    id: 'price-impact',
    section: 'market',
    term: 'Price impact',
    definition:
      'The market is not bottomless. Buy or sell a lot and the price moves against you: buying pushes the ask up, selling pushes the bid down. The first 80 cases of each resource trade at the normal price with a Pantry (more with bigger warehouses), and your recent volume fades by half every night. Small businesses never notice; a big factory does.',
  },
  {
    id: 'market-depth',
    section: 'market',
    term: 'Market depth',
    definition:
      'How many cases you can trade at the normal price before the market reacts. The bulk buttons show what the amount you chose really costs, and the average price when the price has moved. Net worth and your score value stock at the plain bid and ignore price impact.',
  },
  {
    id: 'bulk',
    icon: 'expand',
    section: 'market',
    term: 'Buy or sell amount',
    definition:
      'Choose how many cases each Buy or Sell click moves. If you cannot afford or fit the full amount, it trades as many as it can.',
  },

  {
    id: 'lemon',
    resource: 'lemon',
    section: 'resources',
    term: 'Lemon, sugar and cups',
    definition:
      'Ingredients that stay in your warehouse until they are used or sold. Each has its own warehouse with its own capacity.',
  },
  {
    id: 'ice',
    resource: 'ice',
    section: 'resources',
    term: 'Ice',
    definition:
      'Ice melts. Whatever the day’s production does not use is gone at the end of the day, so only buy the ice you will use. The header shows how much will melt. A freezer upgrade keeps some for one more night.',
  },
  {
    id: 'lemonade',
    resource: 'lemonade',
    section: 'resources',
    term: 'Lemonade',
    definition:
      'The product. It is made at the end of each day and sold on the market like anything else, and it needs warehouse space too.',
  },

  {
    id: 'warehouse',
    icon: 'warehouse',
    section: 'facilities',
    term: 'Warehouses',
    definition:
      'Each resource has its own warehouse, and each building holds a set number of cases. You cannot buy more than you can store.',
  },
  {
    id: 'production',
    icon: 'factory',
    section: 'facilities',
    term: 'Production',
    definition:
      'Makes lemonade at the end of each day, up to a set number. It is also limited by your scarcest ingredient and by space for lemonade.',
  },
  {
    id: 'expand',
    icon: 'expand',
    section: 'facilities',
    term: 'Expand and upgrade',
    definition:
      'Expand adds one more building. Upgrade moves every building of that type up a tier, which makes each one bigger. There is a cap on both.',
  },
  {
    id: 'upkeep',
    icon: 'coin',
    section: 'facilities',
    term: 'Upkeep',
    definition:
      'Every building costs money each day, and bigger tiers cost more. It is charged when you end the day.',
  },
  {
    id: 'sell-building',
    icon: 'upgrade',
    section: 'facilities',
    term: 'Selling a building',
    definition:
      'Returns half of its build cost, and upgrades are not refunded. You must keep one of each, and your stock still has to fit afterwards.',
  },

  {
    id: 'end-day',
    icon: 'end-day',
    section: 'basics',
    term: 'Ending the day',
    definition:
      'In order: lemonade is made, leftover ice melts, upkeep is paid, then the day advances and prices and events update. A report shows what changed.',
  },
  {
    id: 'achievements',
    icon: 'trend-up',
    section: 'basics',
    term: 'Achievements',
    definition:
      'Goals such as a net worth milestone or a long streak. They are cosmetic: they never change how the game plays or your score. Unlocked ones stay with your account, and the Awards page lists them all, with progress. Hidden ones show as ??? until you unlock them.',
  },
  {
    id: 'day-100-board',
    icon: 'calendar',
    section: 'basics',
    term: 'Best by day 100',
    definition:
      'A second scoreboard that ranks your net worth on the day you arrive at day 100, so a short, sharp run and a long one compete on equal terms. A run that ends before day 100 is not on it.',
  },
  {
    id: 'projection',
    icon: 'factory',
    section: 'basics',
    term: 'The projection',
    definition:
      'The header previews the next end of day: how much lemonade will be made, how much ice will melt, and what is limiting production.',
  },
  {
    id: 'production-plan',
    icon: 'upgrade',
    section: 'upgrades',
    term: 'The production plan',
    definition:
      'Each night production works down your plan: every recipe takes what is left of the shared daily capacity, the ingredients you hold and the space for its product. A target of 0 means as much as possible. Learn recipes on the Production page. Fresh goods such as limes and strawberries spoil after a few days, oldest first, so the header warns you before they go off.',
  },
  {
    id: 'upgrades',
    icon: 'upgrade',
    section: 'upgrades',
    term: 'Upgrades',
    definition:
      'One-time purchases that change a rule: they cannot be sold and are not counted in your net worth. Some have a small daily upkeep, paid with your buildings’. Open them from the Upgrades link. Later eras unlock more.',
  },
  {
    id: 'empire',
    icon: 'upgrade',
    section: 'upgrades',
    term: 'Empire and eras',
    definition:
      'Territories are new markets. Entering one costs money and adds a daily hub upkeep, and the highest territory you hold sets your era, which unlocks bigger buildings and upgrades. Each has rival businesses: buy them out, or run a campaign to win share. Open it from the Empire link.',
  },
  {
    id: 'freezer',
    icon: 'warehouse',
    section: 'upgrades',
    term: 'Freezer',
    definition:
      'Keeps fresh ice for one extra night: 20 cases with a chest freezer, 150 with a walk-in, 1,200 with a cold storage wing. The oldest ice is used first, and ice that is still left after the next day’s production melts. The header shows how much is kept and how much will melt.',
  },
  {
    id: 'forecast',
    icon: 'calendar',
    section: 'upgrades',
    term: 'Forecast',
    definition:
      'A weather radio shows tomorrow’s weather event, and the almanac shows every market event three days ahead. The forecast is exact: it is the event that will actually start, not a guess.',
  },
  {
    id: 'events',
    icon: 'event',
    section: 'events',
    term: 'Events',
    definition:
      'Now and then something shakes up the market for a few days. The multiplier applies on top of the normal price.',
  },
];

export interface HelpEvent {
  name: string;
  effects: { resource: Resource; multiplier: number }[];
  days: number;
}

// Mirrors the event table in the backend Config (config.go).
export const HELP_EVENTS: HelpEvent[] = [
  {
    name: 'Heat wave',
    effects: [
      { resource: 'lemonade', multiplier: 1.4 },
      { resource: 'ice', multiplier: 1.3 },
    ],
    days: 2,
  },
  { name: 'Rainy week', effects: [{ resource: 'lemonade', multiplier: 0.75 }], days: 3 },
  { name: 'Lemon blight', effects: [{ resource: 'lemon', multiplier: 1.7 }], days: 3 },
  { name: 'Sugar glut', effects: [{ resource: 'sugar', multiplier: 0.7 }], days: 2 },
  { name: 'Holiday', effects: [{ resource: 'lemonade', multiplier: 1.35 }], days: 1 },
  { name: 'Cup shortage', effects: [{ resource: 'cup', multiplier: 1.5 }], days: 2 },
];
