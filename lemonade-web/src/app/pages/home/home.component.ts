import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { SessionService } from '../../core/session.service';
import { IconComponent } from '../../shared/icon/icon.component';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [RouterLink, IconComponent],
  templateUrl: './home.component.html',
  styleUrl: './home.component.scss',
})
export class HomeComponent {
  protected readonly username = inject(SessionService).username;
}
