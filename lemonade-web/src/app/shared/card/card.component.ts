import { Component, input } from '@angular/core';

/** Titled surface; project header actions with the `cardActions` attribute. */
@Component({
  selector: 'app-card',
  standalone: true,
  templateUrl: './card.component.html',
  styleUrl: './card.component.scss',
})
export class CardComponent {
  readonly title = input.required<string>();
}
