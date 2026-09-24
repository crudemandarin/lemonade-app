import { Component, computed, input } from '@angular/core';

/** Pretty-prints a value as JSON; strings are shown as-is. */
@Component({
  selector: 'app-json-view',
  standalone: true,
  templateUrl: './json-view.component.html',
  styleUrl: './json-view.component.scss',
})
export class JsonViewComponent {
  readonly value = input<unknown>();

  protected readonly text = computed(() => {
    const value = this.value();
    if (value === null || value === undefined || value === '') {
      return null;
    }
    return typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  });
}
