/** Icon names for catalog entries, keyed by the catalog key (files live in assets/ui). */
export const upgradeIcon = (key: string): string => `upgrade-${key}`;
export const territoryIcon = (key: string): string => `territory-${key}`;
export const storageIcon = (storageClass: string): string => `storage-${storageClass}`;

const TERRITORY_RIVAL_SIZE: Record<string, string> = {
  neighborhood: 'stand',
  city: 'shop',
  region: 'shop',
  nation: 'corp',
  world: 'corp',
};

/** Rivals that read as a smaller business than the rest of their territory. */
const RIVAL_SIZE_OVERRIDE: Record<string, string> = {
  main_st_squeeze: 'stand',
  zest_express: 'stand',
};

/** One of three rival icons (stand, shop, corp), chosen by the rival's territory and key. */
export function rivalIcon(key: string, territory: string): string {
  return `rival-${RIVAL_SIZE_OVERRIDE[key] ?? TERRITORY_RIVAL_SIZE[territory] ?? 'shop'}`;
}
