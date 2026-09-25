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
    fixture.componentRef.setInput('projection', {
      lemonadeToProduce: 8,
      iceToMelt: 3,
      limitedBy: 'sugar',
    });
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

  describe('projection', () => {
    const text = () => el.querySelector('.projection')!.textContent!.replace(/\s+/g, ' ');
    const set = (p: object) => {
      fixture.componentRef.setInput('projection', p);
      fixture.detectChanges();
    };

    it('shows lemonade made and ice melting', () => {
      expect(text()).toContain('Makes 8 lemonade');
      expect(text()).toContain('3 ice will melt');
    });

    it('names the limiting factor in the tooltip and aria-label', () => {
      const p = el.querySelector('.projection')!;
      expect(p.getAttribute('title')).toBe('Limited by sugar');
      expect(p.getAttribute('aria-label')).toBe(
        'End of day makes 8 lemonade, 3 ice will melt. Limited by sugar',
      );
    });

    it('words production and space limits', () => {
      set({ lemonadeToProduce: 10, iceToMelt: 0, limitedBy: 'production' });
      expect(el.querySelector('.projection')!.getAttribute('title')).toBe(
        'Limited by production capacity',
      );
      set({ lemonadeToProduce: 2, iceToMelt: 0, limitedBy: 'space' });
      expect(el.querySelector('.projection')!.getAttribute('title')).toBe(
        'Limited by lemonade storage space',
      );
    });

    it('hides the ice figure when none will melt', () => {
      set({ lemonadeToProduce: 5, iceToMelt: 0, limitedBy: '' });
      expect(text()).toContain('Makes 5 lemonade');
      expect(text()).not.toContain('melt');
    });
  });
});
