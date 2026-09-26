import { Component, OnInit, computed, inject, signal } from '@angular/core';

import { PlanRow, Recipe } from '../../core/api.models';
import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';
import { resourceLabel } from '../../core/resources';
import { RunNavComponent } from '../../shared/run-nav/run-nav.component';
import { CardComponent } from '../../shared/card/card.component';
import { ConfirmDialogComponent } from '../../shared/confirm-dialog/confirm-dialog.component';
import { HelpLinkComponent } from '../../shared/help/help-link.component';
import { formatMoney, MoneyPipe } from '../../shared/money.pipe';

/**
 * Recipes and the production plan. Recipes are learned once for a fee (era-gated). The plan is
 * the ordered list production works through each night: every row takes what is left of the
 * shared daily capacity, the input stock and the space for its output; a target of 0 means
 * as much as possible.
 */
@Component({
  selector: 'app-production',
  standalone: true,
  imports: [RunNavComponent, CardComponent, ConfirmDialogComponent, HelpLinkComponent, MoneyPipe],
  templateUrl: './production.component.html',
  styleUrl: './production.component.scss',
})
export class ProductionComponent implements OnInit {
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;
  protected readonly label = resourceLabel;

  protected readonly confirming = signal<Recipe | null>(null);
  /** The plan being edited; null means it matches what the server has. */
  protected readonly draft = signal<PlanRow[] | null>(null);

  protected readonly capital = computed(() => this.store.game()?.capital ?? 0);
  protected readonly active = computed(() => (this.store.game()?.status ?? 'active') === 'active');
  protected readonly recipes = computed(() => this.store.game()?.recipes ?? []);
  protected readonly known = computed(() => this.recipes().filter((r) => r.state === 'known'));
  /** Only recipes that can be bought now: a locked one stays hidden until its era. */
  protected readonly learnable = computed(() =>
    this.recipes().filter((r) => r.state === 'available'),
  );
  protected readonly moreToCome = computed(() => this.recipes().some((r) => r.state === 'locked'));
  protected readonly rows = computed(() => this.draft() ?? this.store.game()?.plan ?? []);
  protected readonly dirty = computed(() => this.draft() !== null);
  protected readonly unused = computed(() =>
    this.known().filter((r) => !this.rows().some((row) => row.recipe === r.key)),
  );
  protected readonly projection = computed(() => this.store.game()?.projection.plan ?? []);

  ngOnInit(): void {
    if (!this.store.game()) {
      void this.store.load();
    }
  }

  protected name(key: string): string {
    return this.recipes().find((r) => r.key === key)?.name ?? key;
  }

  protected exact(value: number): string {
    return formatMoney(value);
  }

  protected inputs(r: Recipe): string {
    return r.inputs
      .map((i) => (i.qty > 1 ? `${i.qty} ${this.label(i.resource)}` : this.label(i.resource)))
      .join(', ');
  }

  protected canLearn(r: Recipe): boolean {
    return (
      r.state === 'available' &&
      this.active() &&
      this.online() &&
      !this.store.loading() &&
      this.capital() >= r.learnCost
    );
  }

  protected async learn(r: Recipe): Promise<void> {
    this.confirming.set(null);
    await this.store.learnRecipe(r.key);
  }

  private edit(change: (rows: PlanRow[]) => void): void {
    const rows = this.rows().map((r) => ({ ...r }));
    change(rows);
    this.draft.set(rows);
  }

  protected add(key: string): void {
    if (key) this.edit((rows) => rows.push({ recipe: key, target: 0 }));
  }

  protected remove(i: number): void {
    this.edit((rows) => rows.splice(i, 1));
  }

  protected move(i: number, by: -1 | 1): void {
    this.edit((rows) => {
      const [row] = rows.splice(i, 1);
      rows.splice(i + by, 0, row);
    });
  }

  protected setTarget(i: number, event: Event): void {
    const value = Math.max(0, Math.floor(Number((event.target as HTMLInputElement).value) || 0));
    this.edit((rows) => (rows[i].target = value));
  }

  protected reset(): void {
    this.draft.set(null);
  }

  protected async save(): Promise<void> {
    const rows = this.draft();
    if (!rows) return;
    await this.store.setPlan(rows);
    if (this.store.error() === null) this.draft.set(null);
  }

  protected madeTonight(key: string): number | null {
    if (this.dirty()) return null;
    return this.projection().find((p) => p.recipe === key)?.cases ?? null;
  }
}
