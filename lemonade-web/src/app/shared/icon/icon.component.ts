import { Component, computed, input } from '@angular/core';

/** Line icons shipped as files in src/assets/ui/<name>.svg. */
const ASSET_ICONS = [
  'calendar',
  'coin',
  'end-day',
  'event',
  'expand',
  'factory',
  'offline',
  'sad-face',
  'trend-down',
  'trend-flat',
  'trend-up',
  'upgrade',
  'warehouse',
] as const;

// Icons with no asset yet, drawn inline in the same style (24×24, 2px stroke).
// Replace with assets/ui/<name>.svg when they exist.
const PATH_ICONS = {
  user: 'M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z',
  'log-out': 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9',
  alert: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20ZM12 8v4M12 16h.01',
  x: 'M18 6 6 18M6 6l12 12',
} as const;

export type IconName = (typeof ASSET_ICONS)[number] | keyof typeof PATH_ICONS;

/**
 * Small line icon that inherits the current text color. Asset icons are applied
 * as a CSS mask over `currentColor`, which an <img> could not do.
 */
@Component({
  selector: 'app-icon',
  standalone: true,
  templateUrl: './icon.component.html',
  styleUrl: './icon.component.scss',
})
export class IconComponent {
  readonly name = input.required<IconName>();
  readonly size = input(16);

  protected readonly path = computed(() => {
    const name = this.name();
    return name in PATH_ICONS ? PATH_ICONS[name as keyof typeof PATH_ICONS] : null;
  });
  protected readonly maskUrl = computed(() => `url(assets/ui/${this.name()}.svg)`);
}
