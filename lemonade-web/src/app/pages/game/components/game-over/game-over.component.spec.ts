import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GameOverComponent } from './game-over.component';

describe('GameOverComponent', () => {
  let fixture: ComponentFixture<GameOverComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(GameOverComponent);
    fixture.componentRef.setInput('day', 12);
    fixture.componentRef.setInput('capital', 0);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  it('shows the final day and capital', () => {
    expect(el.textContent).toContain('Bankrupt on day 12');
    expect(el.textContent).toContain('Final capital: $0.');
  });

  it('emits newGame', () => {
    let count = 0;
    fixture.componentInstance.newGame.subscribe(() => count++);

    el.querySelector<HTMLButtonElement>('.btn-primary')!.click();

    expect(count).toBe(1);
  });

  it('disables New game when disabled', () => {
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();
    expect(el.querySelector<HTMLButtonElement>('.btn-primary')!.disabled).toBeTrue();
  });
});
