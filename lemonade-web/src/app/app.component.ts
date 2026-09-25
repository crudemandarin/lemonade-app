import { Component, computed, inject } from '@angular/core';
import { Router, RouterOutlet } from '@angular/router';

import { AuthService } from './core/auth.service';
import { GameStore } from './core/game.store';
import { SessionService } from './core/session.service';
import { EventBackdropComponent } from './shared/event-backdrop/event-backdrop.component';
import { NavBarComponent } from './shared/nav-bar/nav-bar.component';
import { OfflineBannerComponent } from './shared/offline-banner/offline-banner.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, NavBarComponent, OfflineBannerComponent, EventBackdropComponent],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss',
})
export class AppComponent {
  private readonly router = inject(Router);
  private readonly store = inject(GameStore);
  protected readonly auth = inject(AuthService);
  protected readonly username = inject(SessionService).username;
  protected readonly events = computed(() =>
    this.store.game()?.status !== 'active' ? [] : (this.store.game()?.events ?? []),
  );

  protected logOut(): void {
    this.store.signOut();
    this.router.navigateByUrl('/');
  }
}
