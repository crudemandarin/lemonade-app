import { TestBed } from '@angular/core/testing';

import { OnlineService } from './online.service';

describe('OnlineService', () => {
  it('starts from navigator.onLine', () => {
    expect(TestBed.inject(OnlineService).online()).toBe(navigator.onLine);
  });

  it('follows offline and online events', () => {
    const service = TestBed.inject(OnlineService);

    window.dispatchEvent(new Event('offline'));
    expect(service.online()).toBeFalse();

    window.dispatchEvent(new Event('online'));
    expect(service.online()).toBeTrue();
  });
});
