import { commodities } from './testing/fixtures';
import { catalogOrder, resourceIcon, resourceLabel, seriesColor } from './resources';

describe('resource lookups', () => {
  it('labels known commodities and falls back for new ones', () => {
    expect(resourceLabel('cup')).toBe('Cups');
    expect(resourceLabel('black_tea')).toBe('Black tea');
  });

  it('gives unknown commodities a neutral colour and a generic icon', () => {
    expect(seriesColor('black_tea')).toBe('var(--series-black_tea, var(--series-other))');
    expect(resourceIcon('lemon')).toBe('assets/resources/lemon.svg');
    expect(resourceIcon('lime')).toBe('assets/resources/lime.svg');
    expect(resourceLabel('honey_lemonade')).toBe('Honey lemonade');
    expect(resourceIcon('black_tea')).toBe('assets/resources/generic.svg');
  });

  it('orders by the catalog, or the original five before one is known', () => {
    expect(catalogOrder(commodities())).toEqual(['lemon', 'sugar', 'ice', 'cup', 'lemonade']);
    expect(catalogOrder([{ ...commodities()[4], key: 'limeade' }])).toEqual(['limeade']);
    expect(catalogOrder([])).toEqual(['lemon', 'sugar', 'ice', 'cup', 'lemonade']);
  });
});
