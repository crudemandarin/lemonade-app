import { Commodity, Resource } from './api.models';

/** The original five, in catalog order: the fallback before a view has loaded. */
export const RESOURCE_ORDER: Resource[] = ['lemon', 'sugar', 'ice', 'cup', 'lemonade'];

export const RESOURCE_LABELS: Record<Resource, string> = {
  lemon: 'Lemon',
  sugar: 'Sugar',
  ice: 'Ice',
  cup: 'Cups',
  lemonade: 'Lemonade',
};

/** A display label for any commodity key; unknown keys read as their key, capitalised. */
export function resourceLabel(r: Resource): string {
  return RESOURCE_LABELS[r] ?? r.charAt(0).toUpperCase() + r.slice(1).replace(/_/g, ' ');
}

/** The chart colour for a commodity; one without its own series colour gets a neutral one. */
export function seriesColor(r: Resource): string {
  return `var(--series-${r}, var(--series-other))`;
}

/** The icon for a commodity; one without its own icon gets a generic crate. */
export function resourceIcon(r: Resource): string {
  return RESOURCE_LABELS[r] ? `assets/resources/${r}.svg` : 'assets/resources/generic.svg';
}

/** Commodity keys in catalog order, or the original five when no catalog is known yet. */
export function catalogOrder(commodities: readonly Commodity[] | undefined): Resource[] {
  return commodities && commodities.length ? commodities.map((c) => c.key) : RESOURCE_ORDER;
}
