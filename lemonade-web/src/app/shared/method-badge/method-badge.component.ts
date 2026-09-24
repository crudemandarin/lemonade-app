import { Component, input } from '@angular/core';

/** Colored HTTP method label (GET, POST, PUT, DELETE). */
@Component({
  selector: 'app-method-badge',
  standalone: true,
  templateUrl: './method-badge.component.html',
  styleUrl: './method-badge.component.scss',
})
export class MethodBadgeComponent {
  readonly method = input.required<string>();
}
