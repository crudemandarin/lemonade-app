import { Resource } from './api.models';

/** Resources in the order the API reports stock in each timeline point. */
export const RESOURCE_ORDER: Resource[] = ['lemon', 'sugar', 'ice', 'cup', 'lemonade'];

export const RESOURCE_LABELS: Record<Resource, string> = {
  lemon: 'Lemon',
  sugar: 'Sugar',
  ice: 'Ice',
  cup: 'Cups',
  lemonade: 'Lemonade',
};
