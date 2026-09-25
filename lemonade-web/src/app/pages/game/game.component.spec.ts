import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { newGameView, timelinePoint } from '../../core/testing/fixtures';
import { GameComponent } from './game.component';

describe('GameComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<GameComponent>;

  async function render(overrides = {}) {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    fixture = TestBed.createComponent(GameComponent);
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
    http.expectOne('/api/game').flush(newGameView(overrides));
    await new Promise((resolve) => setTimeout(resolve)); // let the store finish loading
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  }

  afterEach(() => http.verify());

  it('shows the timeline charts on the game page while playing', async () => {
    const el = await render();
    expect(el.querySelector('app-timeline-charts')).not.toBeNull();
    expect(el.querySelector('app-game-over')).toBeNull();
  });

  it('shows the report, with the charts, once bankrupt', async () => {
    const el = await render({
      status: 'bankrupt',
      timeline: [timelinePoint(), timelinePoint({ kind: 'end_day' })],
    });
    expect(el.querySelector('app-game-over app-timeline-charts')).not.toBeNull();
    expect(el.querySelectorAll('app-timeline-charts').length).toBe(1);
  });

  it('shows the same result screen after giving up', async () => {
    const el = await render({ status: 'gave_up', day: 6 });
    expect(el.querySelector('app-game-over h1')!.textContent).toContain('You called it on day 6');
    expect(el.querySelector('.dashboard')).toBeNull();
  });

  describe('give up', () => {
    async function open() {
      const el = await render();
      el.querySelector<HTMLButtonElement>('.give-up button')!.click();
      fixture.detectChanges();
      return el;
    }

    it('is a quiet secondary button, never a second primary', async () => {
      const el = await render();
      expect(el.querySelectorAll('.btn-primary').length).toBe(1);
      expect(el.querySelector('.give-up .btn-primary')).toBeNull();
    });

    it('asks first, showing the score it would record, and does nothing on cancel', async () => {
      const el = await open();
      const dialog = el.querySelector('[role=alertdialog]')!;
      expect(dialog.textContent).toContain('Give up on day 1?');
      expect(dialog.textContent).toContain('$1,500');

      dialog.querySelector<HTMLButtonElement>('.cancel')!.click();
      fixture.detectChanges();
      expect(el.querySelector('[role=alertdialog]')).toBeNull();
      http.expectNone('/api/game/give-up');
    });

    it('posts to give-up once confirmed and shows the result screen', async () => {
      const el = await open();
      el.querySelector<HTMLButtonElement>('[role=alertdialog] .danger')!.click();
      http.expectOne('/api/game/give-up').flush(newGameView({ status: 'gave_up', day: 1 }));
      await new Promise((resolve) => setTimeout(resolve));
      fixture.detectChanges();

      expect(el.querySelector('app-game-over')).not.toBeNull();
      expect(el.querySelector('[role=alertdialog]')).toBeNull();
    });
  });
});
