import { Component, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { AuthService } from '../../core/auth.service';
import { OnlineService } from '../../core/online.service';
import { SessionService } from '../../core/session.service';

const MESSAGES: Record<string, string> = {
  already_linked:
    'That Google account already plays as another username. Log out, then sign in with Google to use it.',
  already_claimed: 'Someone secured this username first, so it cannot be linked.',
  unknown_username: 'This username no longer exists.',
  rate_limited: 'Too many attempts. Try again in a few minutes.',
};

/** Optional: link a Google account so that only it can play this username. */
@Component({
  selector: 'app-secure',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './secure.component.html',
  styleUrl: './secure.component.scss',
})
export class SecureComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  protected readonly online = inject(OnlineService).online;
  protected readonly username = inject(SessionService).username;
  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);

  constructor() {
    // A redirect sign-in that failed on its way back leaves its reason here.
    const code = this.auth.linkError();
    if (code) {
      this.error.set(MESSAGES[code] ?? 'Linking Google did not work. Try again.');
    }
  }

  protected async link(): Promise<void> {
    this.error.set(null);
    this.busy.set(true);
    try {
      await this.auth.linkGoogle();
      if (this.auth.secured()) {
        await this.router.navigateByUrl('/game');
      }
    } catch (err) {
      const code = this.auth.linkError();
      this.error.set((code && MESSAGES[code]) ?? 'Linking Google did not work. Try again.');
      void err;
    } finally {
      this.busy.set(false);
    }
  }
}
