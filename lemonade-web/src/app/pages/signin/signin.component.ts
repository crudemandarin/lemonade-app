import { Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { AuthService } from '../../core/auth.service';
import { GameStore } from '../../core/game.store';
import { OnlineService } from '../../core/online.service';

/**
 * Play on a username alone, or sign in with Google (optional; it is how a secured
 * account is played). One primary button: the username form.
 */
@Component({
  selector: 'app-signin',
  standalone: true,
  templateUrl: './signin.component.html',
  styleUrl: './signin.component.scss',
})
export class SigninComponent {
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  protected readonly store = inject(GameStore);
  protected readonly online = inject(OnlineService).online;
  protected readonly googleAvailable = this.auth.googleAvailable;
  protected readonly fieldError = signal<string | null>(null);
  protected readonly googleBusy = signal(false);
  protected readonly googleError = signal<string | null>(null);
  /** Sent here because a guest name was secured with Google from another device. */
  protected readonly wasSecured =
    inject(ActivatedRoute).snapshot.queryParamMap.get('reason') === 'secured';
  /** Local validation first, then the server's message. */
  protected readonly message = computed(() => this.fieldError() ?? this.store.error());

  constructor() {
    this.store.clearError();
  }

  protected async submit(event: Event, raw: string): Promise<void> {
    event.preventDefault();
    const username = raw.trim().toLowerCase();
    if (!username) {
      this.fieldError.set('Enter a username');
      return;
    }
    if (!/^[\x21-\x7e]{3,40}$/.test(username)) {
      this.fieldError.set('Use 3 to 40 letters, numbers or symbols, with no spaces');
      return;
    }
    this.fieldError.set(null);
    if (await this.store.signIn(username)) {
      await this.router.navigateByUrl('/game');
    }
  }

  protected async google(): Promise<void> {
    this.googleError.set(null);
    this.googleBusy.set(true);
    try {
      await this.auth.signInWithGoogle();
      // A redirect sign-in leaves the page; a popup signs in right here.
      if (this.auth.firebaseUser()) {
        const hasProfile = await this.auth.loadProfile();
        await this.router.navigateByUrl(hasProfile ? '/game' : '/signin/username');
      }
    } catch {
      this.googleError.set('Google sign-in did not work. Try again.');
    } finally {
      this.googleBusy.set(false);
    }
  }
}
