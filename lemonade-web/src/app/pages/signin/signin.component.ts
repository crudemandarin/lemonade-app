import { Component, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';

import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';

@Component({
  selector: 'app-signin',
  standalone: true,
  templateUrl: './signin.component.html',
  styleUrl: './signin.component.scss',
})
export class SigninComponent {
  private readonly router = inject(Router);
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;
  protected readonly fieldError = signal<string | null>(null);
  /** Local validation first, then the server's message. */
  protected readonly message = computed(() => this.fieldError() ?? this.store.error());

  constructor() {
    this.store.clearError();
  }

  protected async submit(event: Event, raw: string): Promise<void> {
    event.preventDefault();
    const username = raw.trim();
    if (!username) {
      this.fieldError.set('Enter a username');
      return;
    }
    this.fieldError.set(null);
    if (await this.store.signIn(username)) {
      await this.router.navigateByUrl('/game');
    }
  }
}
