import { Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';

import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'app-nav-bar',
  standalone: true,
  imports: [RouterLink, IconComponent],
  templateUrl: './nav-bar.component.html',
  styleUrl: './nav-bar.component.scss',
})
export class NavBarComponent {
  readonly username = input<string | null>(null);
  readonly logOut = output<void>();
}
