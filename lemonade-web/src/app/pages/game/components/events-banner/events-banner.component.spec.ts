import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GameEvent } from '../../../../core/api.models';
import { EventsBannerComponent } from './events-banner.component';

const heatWave: GameEvent = {
  key: 'heat_wave',
  name: 'Heat Wave',
  description: 'Scorching days.',
  multipliers: { lemonade: 1.4, ice: 1.3 },
  daysLeft: 2,
};

describe('EventsBannerComponent', () => {
  let fixture: ComponentFixture<EventsBannerComponent>;

  function render(events: GameEvent[]): HTMLElement {
    fixture = TestBed.createComponent(EventsBannerComponent);
    fixture.componentRef.setInput('events', events);
    fixture.detectChanges();
    return fixture.nativeElement;
  }

  it('shows nothing when no event is active', () => {
    expect(render([]).querySelectorAll('.event').length).toBe(0);
  });

  it('summarizes an event with its effects and remaining days', () => {
    const el = render([heatWave]);
    expect(el.querySelector('.event')?.textContent).toContain(
      'Heat Wave: lemonade x1.4 and ice x1.3 for 2 more days',
    );
  });

  it('says "1 more day" for the last day and keeps the description as a tooltip', () => {
    const el = render([{ ...heatWave, multipliers: { lemonade: 1.35 }, daysLeft: 1 }]);
    const line = el.querySelector('.event')!;
    expect(line.textContent).toContain('for 1 more day');
    expect(line.textContent).not.toContain('1 more days');
    expect(line.getAttribute('title')).toBe('Scorching days.');
  });

  it('lists every active event', () => {
    const blight: GameEvent = {
      key: 'lemon_blight',
      name: 'Lemon Blight',
      description: '',
      multipliers: { lemon: 1.7 },
      daysLeft: 3,
    };
    expect(render([heatWave, blight]).querySelectorAll('.event').length).toBe(2);
  });
});
