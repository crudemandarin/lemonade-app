import { Component, input, output } from '@angular/core';

import {
  FacilityType,
  FacilityTypeView,
  ProductionView,
  Resource,
  WarehouseView,
} from '../../../../core/api.models';
import { RESOURCE_LABELS } from '../../../../core/resources';
import { CardComponent } from '../../../../shared/card/card.component';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

/**
 * Warehouse and Production cards. Upgrade applies to every building of a type;
 * expand adds one building (per resource for warehouses). Costs come from the server.
 */
@Component({
  selector: 'app-facilities-panel',
  standalone: true,
  imports: [CardComponent, IconComponent, MoneyPipe],
  templateUrl: './facilities-panel.component.html',
  styleUrl: './facilities-panel.component.scss',
})
export class FacilitiesPanelComponent {
  readonly warehouse = input.required<WarehouseView>();
  readonly production = input.required<ProductionView>();

  readonly expandWarehouse = output<Resource>();
  readonly expandProduction = output<void>();
  readonly upgrade = output<FacilityType>();

  protected readonly labels = RESOURCE_LABELS;

  protected buildings(n: number): string {
    return n === 1 ? '1 building' : `${n} buildings`;
  }

  protected onUpgrade(type: FacilityType, view: FacilityTypeView): void {
    if (view.upgrade) {
      this.upgrade.emit(type);
    }
  }
}
