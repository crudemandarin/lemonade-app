import { Component, input } from '@angular/core';

import { GameEvent, Resource } from '../../../../core/api.models';
import { RESOURCE_LABELS } from '../../../../core/resources';
import { IconComponent } from '../../../../shared/icon/icon.component';

/** Active market events, e.g. "Heat wave: lemonade x1.4 and ice x1.3 for 2 more days". */
@Component({
  selector: 'app-events-banner',
  standalone: true,
  imports: [IconComponent],
  templateUrl: './events-banner.component.html',
  styleUrl: './events-banner.component.scss',
})
export class EventsBannerComponent {
  readonly events = input.required<GameEvent[]>();

  protected summary(event: GameEvent): string {
    const effects = (Object.entries(event.multipliers) as [Resource, number][])
      .map(([resource, m]) => `${RESOURCE_LABELS[resource].toLowerCase()} x${m}`)
      .join(' and ');
    const days = event.daysLeft === 1 ? '1 more day' : `${event.daysLeft} more days`;
    return `${event.name}: ${effects} for ${days}`;
  }
}
