import { fakeAsync, tick } from '@angular/core/testing';

import { TOAST_MS, ToastService } from './toast.service';

describe('ToastService', () => {
  it('shows one line per unlocked achievement and stacks them', () => {
    const toasts = new ToastService();
    toasts.unlocked([
      { key: 'nw_10k', name: 'Five figures', tier: 'bronze' },
      { key: 'day_7', name: 'First week', tier: 'bronze' },
    ]);

    expect(toasts.toasts().map((t) => t.text)).toEqual([
      'Achievement unlocked: Five figures',
      'Achievement unlocked: First week',
    ]);
  });

  it('shows nothing for an empty or missing list', () => {
    const toasts = new ToastService();
    toasts.unlocked([]);
    toasts.unlocked(undefined);

    expect(toasts.toasts()).toEqual([]);
  });

  it('dismisses by itself after a while, and on request', fakeAsync(() => {
    const toasts = new ToastService();
    toasts.show('one');
    toasts.show('two');
    toasts.dismiss(toasts.toasts()[0].id);
    expect(toasts.toasts().map((t) => t.text)).toEqual(['two']);

    tick(TOAST_MS);
    expect(toasts.toasts()).toEqual([]);
  }));
});
