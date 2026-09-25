import { Component, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';

import { apiErrorCode, AuthService } from '../../core/auth.service';
import { apiErrorMessage } from '../../core/api-error';
import { OnlineService } from '../../core/online.service';

const MESSAGES: Record<string, string> = {
  invalid_username: 'Use 3 to 40 letters, numbers or symbols, with no spaces',
  username_taken: 'That username is taken',
  unknown_username: 'No player has that username',
  already_claimed: 'That username already belongs to another Google account',
  rate_limited: 'Too many attempts. Try again in a few minutes',
};

/**
 * First sign-in: choose a username for a new player, or link an existing one from
 * before Google sign-in ("claim"). The username is the only public identity, so it is
 * never prefilled from the Google account.
 */
@Component({
  selector: 'app-username',
  standalone: true,
  templateUrl: './username.component.html',
  styleUrl: './username.component.scss',
})
export class UsernameComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  protected readonly online = inject(OnlineService).online;
  protected readonly claiming = signal(false);
  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly title = computed(() =>
    this.claiming() ? 'Link your username' : 'Choose a username',
  );

  protected toggleClaiming(): void {
    this.claiming.update((v) => !v);
    this.error.set(null);
  }

  protected async submit(event: Event, raw: string): Promise<void> {
    event.preventDefault();
    const username = raw.trim().toLowerCase();
    if (!username) {
      this.error.set('Enter a username');
      return;
    }
    if (!/^[\x21-\x7e]{3,40}$/.test(username)) {
      this.error.set(MESSAGES['invalid_username']);
      return;
    }
    this.error.set(null);
    this.busy.set(true);
    try {
      await (this.claiming() ? this.auth.claim(username) : this.auth.createProfile(username));
      await this.router.navigateByUrl('/game');
    } catch (err) {
      const code = apiErrorCode(err);
      if (code === 'already_linked') {
        // This account already has a player (another tab, or a repeat request).
        await this.auth.loadProfile();
        await this.router.navigateByUrl('/game');
        return;
      }
      this.error.set((code && MESSAGES[code]) ?? apiErrorMessage(err));
    } finally {
      this.busy.set(false);
    }
  }
}
