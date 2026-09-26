import { Component, input } from '@angular/core';

import { ForecastEntry, GameEvent, Resource } from '../../../../core/api.models';
import { resourceLabel } from '../../../../core/resources';
import { HelpLinkComponent } from '../../../../shared/help/help-link.component';
import { IconComponent } from '../../../../shared/icon/icon.component';

/** Active market events, e.g. "Heat wave: lemonade x1.4 and ice x1.3 for 2 more days". */
@Component({
  selector: 'app-events-banner',
  standalone: true,
  imports: [HelpLinkComponent, IconComponent],
  templateUrl: './events-banner.component.html',
  styleUrl: './events-banner.component.scss',
})
export class EventsBannerComponent {
  readonly events = input.required<GameEvent[]>();
  /** Events the player's upgrades let them see coming. */
  readonly forecast = input<ForecastEntry[]>([]);

  protected when(entry: ForecastEntry): string {
    return entry.daysAhead === 1 ? 'Tomorrow' : `In ${entry.daysAhead} days`;
  }

  protected summary(event: GameEvent): string {
    const effects = (Object.entries(event.multipliers) as [Resource, number][])
      .map(([resource, m]) => `${resourceLabel(resource).toLowerCase()} x${m}`)
      .join(' and ');
    if (event.daysLeft === 1) {
      return `${event.name}: ${effects}, today is the last day`;
    }
    return `${event.name}: ${effects} for ${event.daysLeft} more days`;
  }
}
