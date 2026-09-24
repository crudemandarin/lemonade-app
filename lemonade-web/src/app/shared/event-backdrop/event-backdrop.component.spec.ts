import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GameEvent } from '../../core/api.models';
import { EventBackdropComponent } from './event-backdrop.component';

function event(key: string, daysLeft = 1): GameEvent {
  return { key, name: key, description: '', multipliers: {}, daysLeft };
}

describe('EventBackdropComponent', () => {
  let fixture: ComponentFixture<EventBackdropComponent>;

  function render(events: GameEvent[]): HTMLElement {
    fixture.componentRef.setInput('events', events);
    fixture.detectChanges();
    return fixture.nativeElement;
  }

  beforeEach(() => {
    fixture = TestBed.createComponent(EventBackdropComponent);
  });

  it('renders no scene without events', () => {
    expect(render([]).querySelectorAll('.scene').length).toBe(0);
  });

  it('is decorative and ignores pointer input', () => {
    expect(render([]).querySelector('.backdrop')?.getAttribute('aria-hidden')).toBe('true');
  });

  for (const [key, selector] of [
    ['heat_wave', '.heat'],
    ['rainy_week', '.rain'],
    ['lemon_blight', '.blight'],
    ['sugar_glut', '.sugar'],
    ['holiday', '.holiday'],
    ['cup_shortage', '.cups'],
  ]) {
    it(`renders the ${key} scene`, () => {
      expect(render([event(key)]).querySelector(`.scene${selector}`)).not.toBeNull();
    });
  }

  it('ignores unknown event keys', () => {
    expect(render([event('meteor')]).querySelectorAll('.scene').length).toBe(0);
  });

  it('draws at most two scenes, longest-running first', () => {
    const el = render([event('holiday', 1), event('rainy_week', 3), event('heat_wave', 2)]);
    const scenes = Array.from(el.querySelectorAll('.scene'));
    expect(scenes.length).toBe(2);
    expect(scenes[0].classList.contains('rain')).toBeTrue();
    expect(scenes[1].classList.contains('heat')).toBeTrue();
  });

  it('gives particles deterministic in-range variables', () => {
    const vars = (fixture.componentInstance as any).vars(3, 10, 5, 4);
    expect((fixture.componentInstance as any).vars(3, 10, 5, 4)).toEqual(vars);
    expect(parseFloat(vars['--x'])).toBeGreaterThan(0);
    expect(parseFloat(vars['--x'])).toBeLessThan(100);
    expect(parseFloat(vars['--delay'])).toBeLessThanOrEqual(0);
    expect(parseFloat(vars['--j'])).toBeGreaterThanOrEqual(0);
    expect(parseFloat(vars['--j'])).toBeLessThan(1);
  });
});
