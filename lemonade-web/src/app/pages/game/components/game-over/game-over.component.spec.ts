import { ComponentFixture, TestBed } from '@angular/core/testing';

import { gameStats, timelinePoint } from '../../../../core/testing/fixtures';
import { GameOverComponent } from './game-over.component';

describe('GameOverComponent', () => {
  let fixture: ComponentFixture<GameOverComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    fixture = TestBed.createComponent(GameOverComponent);
    fixture.componentRef.setInput('day', 12);
    fixture.componentRef.setInput('capital', 0);
    fixture.componentRef.setInput(
      'stats',
      gameStats({
        peakCapital: 4820,
        peakDay: 9,
        earned: 12345,
        spent: 6789,
        facilitiesBought: 4,
        upgrades: 1,
        facilitySpend: 2300,
        upkeepPaid: 890,
        produced: 210,
        casesBought: 500,
        casesSold: 260,
      }),
    );
    fixture.componentRef.setInput('timeline', [
      timelinePoint(),
      timelinePoint({ kind: 'end_day' }),
    ]);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  it('shows the final day and capital', () => {
    expect(el.textContent).toContain('Bankrupt on day 12');
    expect(el.textContent).toContain('Final capital: $0.');
  });

  it('emits newGame', () => {
    let count = 0;
    fixture.componentInstance.newGame.subscribe(() => count++);

    el.querySelector<HTMLButtonElement>('.btn-primary')!.click();

    expect(count).toBe(1);
  });

  it('disables New game when disabled', () => {
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();
    expect(el.querySelector<HTMLButtonElement>('.btn-primary')!.disabled).toBeTrue();
  });

  it('explains why the game ended', () => {
    expect(el.textContent).toContain("couldn't cover your upkeep");
  });

  describe('game report', () => {
    // Each stat is a <dt>/<dd> pair; read them as "label value" so the assertion matches what a person sees.
    const text = () =>
      Array.from(el.querySelectorAll('.stats > div'))
        .map(
          (row) =>
            `${row.querySelector('dt')!.textContent} ${row.querySelector('dd')!.textContent}`,
        )
        .join(' | ')
        .replace(/\s+/g, ' ');

    it('summarizes the whole game', () => {
      expect(text()).toContain('Peak cash $4,820 on day 9');
      expect(text()).toContain('Earned from sales $12,345');
      expect(text()).toContain('Spent on stock $6,789');
      expect(text()).toContain('Lemonade produced 210');
      expect(text()).toContain('Upkeep paid $890');
    });

    it('counts what was built', () => {
      expect(text()).toContain('4 buildings and 1 upgrade for $2,300');
    });

    it('shows the game timeline charts at full size', () => {
      expect(el.querySelector('app-timeline-charts')).not.toBeNull();
      expect(el.querySelector('app-timeline-charts .capital')).not.toBeNull();
    });

    it('keeps New game as the only primary button', () => {
      expect(el.querySelectorAll('.btn-primary').length).toBe(1);
    });
  });
});
