import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ReportSummary } from '../../../../core/api.models';
import { dayReport } from '../../../../core/testing/fixtures';
import { PastDaysDrawerComponent } from './past-days-drawer.component';

const summary = (day: number): ReportSummary => ({
  day,
  produced: 10,
  capitalBefore: 1000,
  capitalAfter: 970,
  newEvents: [],
  expiredEvents: [],
  bankrupt: false,
});

describe('PastDaysDrawerComponent', () => {
  let fixture: ComponentFixture<PastDaysDrawerComponent>;
  let el: HTMLElement;

  function render(inputs: Record<string, unknown>) {
    fixture = TestBed.createComponent(PastDaysDrawerComponent);
    for (const [name, value] of Object.entries(inputs)) {
      fixture.componentRef.setInput(name, value);
    }
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  const button = (cls: string) => el.querySelector<HTMLButtonElement>(`.${cls}`)!;

  it('says so, and that old days are not kept, when there is no history', () => {
    render({ days: [] });
    expect(el.querySelector('.empty')!.textContent).toContain('No days have ended yet');
    expect(el.querySelector('.empty')!.textContent).toContain("aren't kept");
    expect(el.querySelector('.stepper')).toBeNull();
  });

  it('shows the selected day with a stepper', () => {
    render({
      days: [summary(1), summary(2), summary(3)],
      selected: 2,
      report: dayReport({ day: 2 }),
    });
    expect(el.querySelector('h3')!.textContent).toContain('Day 2 report');
    expect(el.querySelector('app-day-report')).not.toBeNull();
    expect(button('prev').disabled).toBeFalse();
    expect(button('next').disabled).toBeFalse();
  });

  it('steps to the previous and next day, and stops at both ends', () => {
    render({
      days: [summary(1), summary(2), summary(3)],
      selected: 2,
      report: dayReport({ day: 2 }),
    });
    const picked: number[] = [];
    fixture.componentInstance.selectDay.subscribe((d) => picked.push(d));
    button('prev').click();
    button('next').click();
    expect(picked).toEqual([1, 3]);

    fixture.componentRef.setInput('selected', 1);
    fixture.detectChanges();
    expect(button('prev').disabled).toBeTrue();
    fixture.componentRef.setInput('selected', 3);
    fixture.detectChanges();
    expect(button('next').disabled).toBeTrue();
  });

  it('jumps to any day from the list', () => {
    render({
      days: [summary(1), summary(2), summary(3)],
      selected: 3,
      report: dayReport({ day: 3 }),
    });
    const picked: number[] = [];
    fixture.componentInstance.selectDay.subscribe((d) => picked.push(d));
    const select = el.querySelector<HTMLSelectElement>('select')!;
    expect(Array.from(select.options).map((o) => o.textContent!.trim())).toEqual([
      'Day 1',
      'Day 2',
      'Day 3',
    ]);
    select.value = '1';
    select.dispatchEvent(new Event('change'));
    expect(picked).toEqual([1]);
  });

  it('shows loading and error states', () => {
    render({ days: null });
    expect(el.textContent).toContain('Loading');
    fixture.componentRef.setInput('failed', true);
    fixture.detectChanges();
    expect(el.querySelector('[role=alert]')!.textContent).toContain("Couldn't load");
  });

  it('closes from the button and from Escape', () => {
    render({ days: [] });
    let closed = 0;
    fixture.componentInstance.closed.subscribe(() => closed++);
    button('close').click();
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(closed).toBe(2);
  });
});
