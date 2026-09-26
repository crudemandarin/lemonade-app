import { rivalIcon, storageIcon, territoryIcon, upgradeIcon } from './icons';

describe('catalog icons', () => {
  it('builds names from catalog keys', () => {
    expect(upgradeIcon('freezer_1')).toBe('upgrade-freezer_1');
    expect(territoryIcon('city')).toBe('territory-city');
    expect(storageIcon('cold')).toBe('storage-cold');
  });

  it('sizes a rival by territory, with per-rival overrides', () => {
    expect(rivalIcon('lil_lucy', 'neighborhood')).toBe('rival-stand');
    expect(rivalIcon('cafe_citron', 'city')).toBe('rival-shop');
    expect(rivalIcon('main_st_squeeze', 'city')).toBe('rival-stand');
    expect(rivalIcon('global_citrus', 'world')).toBe('rival-corp');
    expect(rivalIcon('new_one', 'moon')).toBe('rival-shop');
  });
});
