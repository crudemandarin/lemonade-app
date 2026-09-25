export type HelpSection = 'basics' | 'market' | 'resources' | 'facilities' | 'events';

export interface HelpTerm {
  id: string;
  section: HelpSection;
  term: string;
  definition: string;
}

export const HELP_SECTIONS: { id: HelpSection; label: string }[] = [
  { id: 'basics', label: 'Basics' },
  { id: 'market', label: 'Market' },
  { id: 'resources', label: 'Resources' },
  { id: 'facilities', label: 'Facilities' },
  { id: 'events', label: 'Day and events' },
];

// Numbers in the prose mirror the backend Config (lemonade-api/internal/domain/config.go):
// Spread (10%), the 50% resale rate and the event table. Update both together.
export const HELP_TERMS: HelpTerm[] = [
  {
    id: 'recipe',
    section: 'basics',
    term: 'The recipe',
    definition:
      'One lemon, one sugar, one ice and one cup make one lemonade. Buy the ingredients low, and sell the lemonade high.',
  },
  {
    id: 'day',
    section: 'basics',
    term: 'Days',
    definition:
      'The game is turn based. Nothing changes until you press End day, so take as long as you like.',
  },
  {
    id: 'capital',
    section: 'basics',
    term: 'Capital',
    definition:
      'The cash you have on hand. You spend it on stock and buildings, and earn it by selling.',
  },
  {
    id: 'net-worth',
    section: 'basics',
    term: 'Net worth',
    definition:
      'Your capital plus the value of your stock, at the price you could sell it for right now.',
  },
  {
    id: 'bankrupt',
    section: 'basics',
    term: 'Going bankrupt',
    definition:
      'Upkeep is due every day. If your cash falls short, stock is sold at the bid to cover it: lemonade first, then lemon, sugar, cups and ice. If that still is not enough, the game is over.',
  },

  {
    id: 'price',
    section: 'market',
    term: 'Price',
    definition:
      'Prices drift a little each day but are pulled back toward a normal level. Events can push them well above or below it.',
  },
  {
    id: 'bid',
    section: 'market',
    term: 'Bid',
    definition: 'What the market pays you when you sell. It sits 10% below the price.',
  },
  {
    id: 'ask',
    section: 'market',
    term: 'Ask',
    definition:
      'What the market charges you when you buy. It sits 10% above the price, so buying and selling straight away always loses a little.',
  },
  {
    id: 'avg-cost',
    section: 'market',
    term: 'Average cost',
    definition:
      'The average you paid per case of what you hold. For lemonade it is what its ingredients cost. The green or red figure is your gain or loss if you sold everything at the bid.',
  },
  {
    id: 'bulk',
    section: 'market',
    term: 'Buy or sell amount',
    definition:
      'Choose how many cases each Buy or Sell click moves. If you cannot afford or fit the full amount, it trades as many as it can.',
  },

  {
    id: 'lemon',
    section: 'resources',
    term: 'Lemon, sugar and cups',
    definition:
      'Ingredients that stay in your warehouse until they are used or sold. Each has its own warehouse with its own capacity.',
  },
  {
    id: 'ice',
    section: 'resources',
    term: 'Ice',
    definition:
      'Ice melts. Whatever the day’s production does not use is gone at the end of the day, so only buy the ice you will use. The header shows how much will melt.',
  },
  {
    id: 'lemonade',
    section: 'resources',
    term: 'Lemonade',
    definition:
      'The product. It is made at the end of each day and sold on the market like anything else, and it needs warehouse space too.',
  },

  {
    id: 'warehouse',
    section: 'facilities',
    term: 'Warehouses',
    definition:
      'Each resource has its own warehouse, and each building holds a set number of cases. You cannot buy more than you can store.',
  },
  {
    id: 'production',
    section: 'facilities',
    term: 'Production',
    definition:
      'Makes lemonade at the end of each day, up to a set number. It is also limited by your scarcest ingredient and by space for lemonade.',
  },
  {
    id: 'expand',
    section: 'facilities',
    term: 'Expand and upgrade',
    definition:
      'Expand adds one more building. Upgrade moves every building of that type up a tier, which makes each one bigger. There is a cap on both.',
  },
  {
    id: 'upkeep',
    section: 'facilities',
    term: 'Upkeep',
    definition:
      'Every building costs money each day, and bigger tiers cost more. It is charged when you end the day.',
  },
  {
    id: 'sell-building',
    section: 'facilities',
    term: 'Selling a building',
    definition:
      'Returns half of its build cost, and upgrades are not refunded. You must keep one of each, and your stock still has to fit afterwards.',
  },

  {
    id: 'end-day',
    section: 'events',
    term: 'Ending the day',
    definition:
      'In order: lemonade is made, leftover ice melts, upkeep is paid, then the day advances and prices and events update. A report shows what changed.',
  },
  {
    id: 'projection',
    section: 'events',
    term: 'The projection',
    definition:
      'The header previews the next end of day: how much lemonade will be made, how much ice will melt, and what is limiting production.',
  },
  {
    id: 'events',
    section: 'events',
    term: 'Events',
    definition:
      'Now and then something shakes up the market for a few days. The multiplier applies on top of the normal price.',
  },
];

export interface HelpEvent {
  name: string;
  effect: string;
  days: number;
}

export const HELP_EVENTS: HelpEvent[] = [
  { name: 'Heat wave', effect: 'Lemonade x1.4, ice x1.3', days: 2 },
  { name: 'Rainy week', effect: 'Lemonade x0.75', days: 3 },
  { name: 'Lemon blight', effect: 'Lemons x1.7', days: 3 },
  { name: 'Sugar glut', effect: 'Sugar x0.7', days: 2 },
  { name: 'Holiday', effect: 'Lemonade x1.35', days: 1 },
  { name: 'Cup shortage', effect: 'Cups x1.5', days: 2 },
];
