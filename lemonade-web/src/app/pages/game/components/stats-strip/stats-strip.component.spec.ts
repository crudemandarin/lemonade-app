import { ComponentFixture, TestBed } from '@angular/core/testing';

import { StatsStripComponent } from './stats-strip.component';

describe('StatsStripComponent', () => {
  let fixture: ComponentFixture<StatsStripComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(StatsStripComponent);
    fixture.componentRef.setInput('day', 4);
    fixture.componentRef.setInput('capital', 1240);
    fixture.componentRef.setInput('upkeepPerDay', 15);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  const endDay = () => el.querySelector<HTMLButtonElement>('.btn-primary')!;

  it('shows day, capital, and upkeep', () => {
    const text = el.textContent!;
    expect(text).toContain('4');
    expect(text).toContain('$1,240');
    expect(text).toContain('$15');
  });

  it('emits endDay', () => {
    let count = 0;
    fixture.componentInstance.endDay.subscribe(() => count++);
    endDay().click();
    expect(count).toBe(1);
  });

  it('disables End day when disabled', () => {
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();
    expect(endDay().disabled).toBeTrue();
  });
});
