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

  it('lists all five prices with lemonade first', () => {
    const labels = Array.from(el.querySelectorAll('.prices dt')).map((n) => n.textContent!.trim());
    expect(labels).toEqual(['Lemonade', 'Lemon', 'Sugar', 'Ice', 'Cups']);
  });

  it('says "no change" for an unchanged price, including lemonade', () => {
    fixture.componentRef.setInput(
      'report',
      dayReport({
        priceChanges: [
          { resource: 'lemon', before: 20, after: 21 },
          { resource: 'lemonade', before: 90, after: 90 },
        ],
      }),
    );
    fixture.detectChanges();
    const rows = Array.from(el.querySelectorAll('.prices dd')).map((n) =>
      n.textContent!.replace(/\s+/g, ' ').trim(),
    );
    expect(rows).toEqual(['$90 · no change', '$20 → $21']);
    expect(el.querySelector('.prices dd.unchanged')).not.toBeNull();
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

  describe('stock sold to cover upkeep', () => {
    function render(overrides: Parameters<typeof dayReport>[0]): string {
      fixture.componentRef.setInput('report', dayReport(overrides));
      fixture.detectChanges();
      return (fixture.nativeElement as HTMLElement).textContent!.replace(/\s+/g, ' ');
    }

    it('shows what was sold and for how much', () => {
      const text = render({ forcedSaleCases: 2, forcedSaleProceeds: 162 });
      expect(text).toContain('Stock sold to cover upkeep');
      expect(text).toContain('2 cases');
      expect(text).toContain('+$162');
    });

    it('uses the singular for one case', () => {
      expect(render({ forcedSaleCases: 1, forcedSaleProceeds: 81 })).toContain('1 case ');
    });

    it('is not shown when cash covered upkeep', () => {
      expect(render({ forcedSaleCases: 0, forcedSaleProceeds: 0 })).not.toContain('sold to cover');
    });
  });
});
