import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { AppComponent } from './app.component';
import { GameStore } from './core/game.store';
import { SessionService } from './core/session.service';
import { newGameView } from './core/testing/fixtures';
import { FakeAuthPort, provideFakeAuth, signInForTest } from './core/testing/fake-auth';

describe('AppComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        provideFakeAuth(new FakeAuthPort()),
      ],
    }).compileComponents();
  });

  afterEach(() => TestBed.inject(SessionService).signOut());

  it('renders the nav bar', () => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('app-nav-bar')).not.toBeNull();
  });

  it('log out clears the session and routes home', () => {
    signInForTest('lemonjoe');
    const session = TestBed.inject(SessionService);
    const navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    fixture.nativeElement.querySelector('.log-out').click();

    expect(session.username()).toBeNull();
    expect(navigate).toHaveBeenCalledWith('/');
  });

  describe('event backdrop', () => {
    const heatWave = {
      key: 'heat_wave',
      name: 'Heat Wave',
      description: '',
      multipliers: {},
      daysLeft: 2,
    };

    async function loadGame(status: 'active' | 'bankrupt') {
      signInForTest('lemonjoe');
      const fixture = TestBed.createComponent(AppComponent);
      const load = TestBed.inject(GameStore).load();
      TestBed.inject(HttpTestingController)
        .expectOne('/api/game')
        .flush(newGameView({ status, events: [heatWave] }));
      await load;
      fixture.detectChanges();
      return fixture.nativeElement as HTMLElement;
    }

    it('draws the active events', async () => {
      expect((await loadGame('active')).querySelector('.scene.heat')).not.toBeNull();
    });

    it('draws nothing once the game is over', async () => {
      expect((await loadGame('bankrupt')).querySelector('.scene')).toBeNull();
    });
  });
});
