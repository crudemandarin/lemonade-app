import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';

import { CardComponent } from './card.component';

@Component({
  standalone: true,
  imports: [CardComponent],
  template: `<app-card title="Market">
    <button cardActions>Refresh</button>
    <p class="body">Body text</p>
  </app-card>`,
})
class HostComponent {}

describe('CardComponent', () => {
  it('renders the title, projects header actions into the header and the rest below it', () => {
    const fixture = TestBed.createComponent(HostComponent);
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;

    expect(el.querySelector('h2')?.textContent).toBe('Market');
    expect(el.querySelector('.card-header button')?.textContent).toBe('Refresh');
    expect(el.querySelector('.card-header .body')).toBeNull();
    expect(el.querySelector('.card > .body')?.textContent).toBe('Body text');
  });
});
