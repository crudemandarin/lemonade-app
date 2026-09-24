import { Component, input, output } from '@angular/core';

import { IconComponent } from '../../../../shared/icon/icon.component';
import { MoneyPipe } from '../../../../shared/money.pipe';

@Component({
  selector: 'app-game-over',
  standalone: true,
  imports: [IconComponent, MoneyPipe],
  templateUrl: './game-over.component.html',
  styleUrl: './game-over.component.scss',
})
export class GameOverComponent {
  readonly day = input.required<number>();
  readonly capital = input.required<number>();
  readonly newGame = output<void>();
}
