import { Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { SessionService } from '../../core/session.service';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './home.component.html',
  styleUrl: './home.component.scss',
})
export class HomeComponent {
  protected readonly username = inject(SessionService).username;
}
