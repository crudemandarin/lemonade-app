import { Component, input, output, signal } from '@angular/core';

import {
  FacilityType,
  FacilityTypeView,
  ProductionView,
  SaleInfo,
  WarehouseView,
} from '../../../../core/api.models';
import { resourceIcon, resourceLabel } from '../../../../core/resources';
import { CardComponent } from '../../../../shared/card/card.component';
import { ConfirmDialogComponent } from '../../../../shared/confirm-dialog/confirm-dialog.component';
import { HelpLinkComponent } from '../../../../shared/help/help-link.component';
import { storageIcon } from '../../../../core/icons';
import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

/** A building sale waiting for the player's confirmation. */
interface PendingSale {
  type: FacilityType;
  class?: string;
  /** e.g. "Pantry" or "Kitchen". */
  building: string;
  value: number;
}

export interface SellFacilityRequest {
  type: FacilityType;
  class?: string;
}

/**
 * Warehouse and Production cards. Upgrade applies to every building of a type;
 * expand adds one building (per storage class for warehouses, pooled across its commodities). Costs come from the server.
 */
@Component({
  selector: 'app-facilities-panel',
  standalone: true,
  imports: [HelpLinkComponent, CardComponent, ConfirmDialogComponent, IconComponent, MoneyPipe],
  templateUrl: './facilities-panel.component.html',
  styleUrl: './facilities-panel.component.scss',
})
export class FacilitiesPanelComponent {
  readonly warehouse = input.required<WarehouseView>();
  readonly production = input.required<ProductionView>();
  /** Disables every action, e.g. while offline. */
  readonly disabled = input(false);

  readonly expandWarehouse = output<string>();
  readonly expandProduction = output<void>();
  readonly upgrade = output<FacilityType>();
  readonly sellFacility = output<SellFacilityRequest>();

  protected readonly pending = signal<PendingSale | null>(null);

  protected readonly label = resourceLabel;
  protected readonly icon = resourceIcon;
  protected readonly storage = storageIcon;

  protected buildings(n: number): string {
    return n === 1 ? '1 building' : `${n} buildings`;
  }

  protected askToSell(sale: PendingSale): void {
    this.pending.set(sale);
  }

  protected confirmSale(): void {
    const sale = this.pending();
    this.pending.set(null);
    if (sale) {
      this.sellFacility.emit({ type: sale.type, class: sale.class });
    }
  }

  /** Why a building cannot be sold, in words; empty when it can. */
  protected blockedReason(sale: SaleInfo): string {
    if (sale.canSell) {
      return '';
    }
    if (sale.sellBlockedReason === 'stock_exceeds_capacity') {
      return `Sell ${sale.casesToSell} ${sale.casesToSell === 1 ? 'case' : 'cases'} first`;
    }
    return 'Keep at least one';
  }

  protected onUpgrade(type: FacilityType, view: FacilityTypeView): void {
    if (view.upgrade) {
      this.upgrade.emit(type);
    }
  }
}
