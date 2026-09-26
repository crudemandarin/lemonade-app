import { TestBed } from '@angular/core/testing';

import { ToastService } from '../../core/toast.service';
import { ToastHostComponent } from './toast-host.component';

describe('ToastHostComponent', () => {
  it('announces toasts in a polite live region and lets them be dismissed', () => {
    const fixture = TestBed.createComponent(ToastHostComponent);
    const toasts = TestBed.inject(ToastService);
    fixture.detectChanges();
    const el: HTMLElement = fixture.nativeElement;

    const region = el.querySelector('[aria-live]')!;
    expect(region.getAttribute('aria-live')).toBe('polite');
    expect(region.getAttribute('role')).toBe('status');
    expect(el.querySelectorAll('.toast').length).toBe(0);

    toasts.unlocked([{ key: 'nw_10k', name: 'Five figures', tier: 'bronze' }]);
    fixture.detectChanges();
    expect(el.querySelector('.toast')!.textContent).toContain('Achievement unlocked: Five figures');

    el.querySelector<HTMLButtonElement>('.close')!.click();
    fixture.detectChanges();
    expect(el.querySelectorAll('.toast').length).toBe(0);
  });
});
