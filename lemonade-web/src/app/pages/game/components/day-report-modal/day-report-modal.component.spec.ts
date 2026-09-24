import { ComponentFixture, TestBed } from '@angular/core/testing';

import { dayReport } from '../../../../core/testing/fixtures';
import { DayReportModalComponent } from './day-report-modal.component';

describe('DayReportModalComponent', () => {
  let fixture: ComponentFixture<DayReportModalComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(DayReportModalComponent);
    fixture.componentRef.setInput(
      'report',
      dayReport({
        newEvents: [
          { key: 'heat_wave', name: 'Heat wave', description: '', multipliers: {}, daysLeft: 2 },
        ],
      }),
    );
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  it('shows the report lines from the DayReport', () => {
    const text = el.textContent!.replace(/\s+/g, ' ');
    expect(text).toContain('Day 4 report');
    expect(text).toContain('+10');
    expect(text).toContain('2 cases');
    expect(text).toContain('-$15');
    expect(text).toContain('$100 → $140');
    expect(text).toContain('Heat wave');
    expect(text).toContain('$1,240 → $1,225');
  });

  it('emits dismiss from the Start day button', () => {
    let dismissed = 0;
    fixture.componentInstance.dismiss.subscribe(() => dismissed++);

    const button = el.querySelector<HTMLButtonElement>('.btn-primary')!;
    expect(button.textContent).toContain('Start day 5');
    button.click();

    expect(dismissed).toBe(1);
  });
});
