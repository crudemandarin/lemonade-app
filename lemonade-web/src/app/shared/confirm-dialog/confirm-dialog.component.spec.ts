import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ConfirmDialogComponent } from './confirm-dialog.component';

describe('ConfirmDialogComponent', () => {
  let fixture: ComponentFixture<ConfirmDialogComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(ConfirmDialogComponent);
    fixture.componentRef.setInput('title', 'Sell a Pantry?');
    fixture.componentRef.setInput('message', 'You get $50 back.');
    fixture.componentRef.setInput('confirmLabel', 'Sell for $50');
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  it('shows the title, message and confirm label, and is never primary', () => {
    expect(el.querySelector('[role=alertdialog]')).not.toBeNull();
    expect(el.textContent).toContain('Sell a Pantry?');
    expect(el.textContent).toContain('You get $50 back.');
    expect(el.querySelector('.danger')!.textContent).toContain('Sell for $50');
    expect(el.querySelector('.btn-primary')).toBeNull();
  });

  it('emits confirmed and cancelled', () => {
    let confirmed = 0;
    let cancelled = 0;
    fixture.componentInstance.confirmed.subscribe(() => confirmed++);
    fixture.componentInstance.cancelled.subscribe(() => cancelled++);

    el.querySelector<HTMLButtonElement>('.danger')!.click();
    el.querySelector<HTMLButtonElement>('.cancel')!.click();

    expect([confirmed, cancelled]).toEqual([1, 1]);
  });
});
