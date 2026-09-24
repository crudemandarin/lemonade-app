import { Component, input, output } from '@angular/core';

import { Resource, ResourceView } from '../../../../core/api.models';
import { RESOURCE_LABELS } from '../../../../core/resources';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

export interface TradeRequest {
  resource: Resource;
  qty: number;
}

/** Stock, prices, and buy/sell controls per resource. Quantity checks happen on the server. */
@Component({
  selector: 'app-market-panel',
  standalone: true,
  imports: [CardComponent, IconComponent, MoneyPipe],
  templateUrl: './market-panel.component.html',
  styleUrl: './market-panel.component.scss',
})
export class MarketPanelComponent {
  readonly resources = input.required<ResourceView[]>();
  readonly buy = output<TradeRequest>();
  readonly sell = output<TradeRequest>();

  protected readonly labels = RESOURCE_LABELS;

  protected trend(row: ResourceView): 'up' | 'down' | 'flat' {
    if (row.previousPrice === null || row.previousPrice === row.price) {
      return 'flat';
    }
    return row.price > row.previousPrice ? 'up' : 'down';
  }

  protected fill(row: ResourceView): number {
    return row.capacity > 0 ? Math.min(100, (row.stock / row.capacity) * 100) : 0;
  }

  protected qty(input: HTMLInputElement): number {
    return Number(input.value);
  }
}
