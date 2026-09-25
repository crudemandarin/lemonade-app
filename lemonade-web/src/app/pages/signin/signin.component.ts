import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';

import { AuthService } from '../../core/auth.service';
import { OnlineService } from '../../core/online.service';

@Component({
  selector: 'app-signin',
  standalone: true,
  templateUrl: './signin.component.html',
  styleUrl: './signin.component.scss',
})
export class SigninComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  protected readonly online = inject(OnlineService).online;
  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);

  protected async signIn(): Promise<void> {
    this.error.set(null);
    this.busy.set(true);
    try {
      await this.auth.signInWithGoogle();
      // A redirect sign-in leaves the page; a popup signs in right here.
      if (this.auth.firebaseUser()) {
        const hasProfile = await this.auth.loadProfile();
        await this.router.navigateByUrl(hasProfile ? '/game' : '/signin/username');
      }
    } catch {
      this.error.set('Google sign-in did not work. Try again.');
    } finally {
      this.busy.set(false);
    }
  }
}
