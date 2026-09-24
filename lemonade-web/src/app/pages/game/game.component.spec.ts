import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { newGameView, timelinePoint } from '../../core/testing/fixtures';
import { GameComponent } from './game.component';

describe('GameComponent', () => {
  let http: HttpTestingController;

  async function render(overrides = {}) {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    const fixture = TestBed.createComponent(GameComponent);
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
});
