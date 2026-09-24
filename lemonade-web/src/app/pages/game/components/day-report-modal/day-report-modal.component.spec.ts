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

  describe('out-of-cash warning', () => {
    function render(overrides: Parameters<typeof dayReport>[0]): HTMLElement {
      fixture.componentRef.setInput('report', dayReport(overrides));
      fixture.detectChanges();
      return fixture.nativeElement;
    }

    it('warns when capital is $0 but the game continues', () => {
      const warning = render({ capitalAfter: 0, bankrupt: false }).querySelector('.warning');
      expect(warning?.textContent).toContain("You're out of cash");
    });

    it('does not warn when there is capital left', () => {
      expect(render({ capitalAfter: 5 }).querySelector('.warning')).toBeNull();
    });

    it('does not warn when the game is over', () => {
      expect(render({ capitalAfter: 0, bankrupt: true }).querySelector('.warning')).toBeNull();
    });
  });
});
