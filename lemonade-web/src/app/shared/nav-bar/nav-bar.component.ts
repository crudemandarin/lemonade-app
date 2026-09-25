import { Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';

import { HelpPanelComponent } from '../help/help-panel.component';
import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'app-nav-bar',
  standalone: true,
  imports: [RouterLink, HelpPanelComponent, IconComponent],
  templateUrl: './nav-bar.component.html',
  styleUrl: './nav-bar.component.scss',
})
export class NavBarComponent {
  readonly username = input<string | null>(null);
  /** The account is protected by Google. */
  readonly secured = input(false);
  /** A guest could link Google (it is configured), so offer to. */
  readonly canSecure = input(false);
  readonly logOut = output<void>();
}
