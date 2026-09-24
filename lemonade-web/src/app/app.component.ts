import { Component, inject } from '@angular/core';
import { Router, RouterOutlet } from '@angular/router';

import { GameStore } from './core/game.store';
import { SessionService } from './core/session.service';
import { NavBarComponent } from './shared/nav-bar/nav-bar.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, NavBarComponent],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss',
})
export class AppComponent {
  private readonly router = inject(Router);
  private readonly store = inject(GameStore);
  protected readonly username = inject(SessionService).username;

  protected logOut(): void {
    this.store.signOut();
    this.router.navigateByUrl('/');
  }
}
