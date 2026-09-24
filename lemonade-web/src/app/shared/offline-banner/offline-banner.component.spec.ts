import { ComponentFixture, TestBed } from '@angular/core/testing';

import { OfflineBannerComponent } from './offline-banner.component';

describe('OfflineBannerComponent', () => {
  let fixture: ComponentFixture<OfflineBannerComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(OfflineBannerComponent);
    el = fixture.nativeElement;
  });

  afterEach(() => window.dispatchEvent(new Event('online')));

  it('is hidden while online', () => {
    window.dispatchEvent(new Event('online'));
    fixture.detectChanges();
    expect(el.querySelector('[role=status]')).toBeNull();
  });

  it('shows a notice while offline', () => {
    window.dispatchEvent(new Event('offline'));
    fixture.detectChanges();
    expect(el.querySelector('[role=status]')?.textContent).toContain('You are offline');
  });
});
