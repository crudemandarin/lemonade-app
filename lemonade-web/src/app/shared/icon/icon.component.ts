import { Component, input } from '@angular/core';

// Stroke paths in the Lucide style (24×24, 2px stroke). The only icon set in the app.
const ICON_PATHS = {
  lemon:
    'M21.66 17.67a1.08 1.08 0 0 1-.04 1.6A12 12 0 0 1 4.73 2.38a1.1 1.1 0 0 1 1.61-.04ZM19.65 15.66A8 8 0 0 1 8.35 4.34M14 10l-5.5 5.5M14 17.85V10H6.15',
  user: 'M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z',
  'log-out': 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9',
  'arrow-right': 'M5 12h14M12 5l7 7-7 7',
  'trend-up': 'm22 7-8.5 8.5-5-5L2 17M16 7h6v6',
  'trend-down': 'm22 17-8.5-8.5-5 5L2 7M16 17h6v-6',
  zap: 'M13 2 3 14h9l-1 8 10-12h-9l1-8Z',
  frown: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20ZM16 16s-1.5-2-4-2-4 2-4 2M9 9h.01M15 9h.01',
  warehouse:
    'M22 8.35V20a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V8.35A2 2 0 0 1 3.26 6.5l8-3.2a2 2 0 0 1 1.48 0l8 3.2A2 2 0 0 1 22 8.35ZM6 18h12M6 14h12M6 10h12v12H6Z',
  factory:
    'M2 20a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8l-7 5V8l-7 5V4a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2ZM17 18h1M12 18h1M7 18h1',
  alert: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20ZM12 8v4M12 16h.01',
  x: 'M18 6 6 18M6 6l12 12',
} as const;

export type IconName = keyof typeof ICON_PATHS;

/** Small stroke icon that inherits the current text color. */
@Component({
  selector: 'app-icon',
  standalone: true,
  templateUrl: './icon.component.html',
  styleUrl: './icon.component.scss',
})
export class IconComponent {
  readonly name = input.required<IconName>();
  readonly size = input(16);

  protected readonly paths = ICON_PATHS;
}
