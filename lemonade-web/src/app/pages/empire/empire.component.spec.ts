import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { ApiService } from '../../core/api.service';
import { EmpireResponse, Territory } from '../../core/api.models';
import { newGameView } from '../../core/testing/fixtures';
import { EmpireComponent } from './empire.component';

function territory(o: Partial<Territory> = {}): Territory {
  return {
    key: 'neighborhood',
    name: 'Neighborhood',
    era: 1,
    entered: true,
    depth: 100,
    entryCost: 0,
    entryShare: 40,
    hubUpkeep: 0,
    buildingCap: 2,
    share: 40,
    reach: 40,
    enterBlocked: 'already_entered',
    campaignDaysLeft: 0,
    campaignBonus: 0,
    campaigns: [],
    rivals: [],
    ...o,
  };
}

const empire = (territories: Territory[]): EmpireResponse => ({
  era: 1,
  eraName: 'Neighborhood',
  hubUpkeep: 0,
  acquisitions: 0,
  reach: 40,
  nextGoal: null,
  territories,
});

describe('EmpireComponent', () => {
  function setup(territories: Territory[], capital = 1000) {
    const api = jasmine.createSpyObj<ApiService>('ApiService', [
      'getGame',
      'empire',
      'enterTerritory',
    ]);
    api.getGame.and.returnValue(of(newGameView({ capital })));
    api.empire.and.returnValue(of(empire(territories)));
    api.enterTerritory.and.returnValue(of(newGameView()));
    TestBed.configureTestingModule({
      imports: [EmpireComponent],
      providers: [{ provide: ApiService, useValue: api }, provideRouter([])],
    });
    const fixture = TestBed.createComponent(EmpireComponent);
    fixture.detectChanges();
    return { api, fixture, el: fixture.nativeElement as HTMLElement };
  }

  it('shows a locked territory with why it cannot be entered', async () => {
    const { fixture, el } = setup([
      territory(),
      territory({
        key: 'city',
        name: 'City',
        entered: false,
        entryCost: 5000,
        enterBlocked: 'insufficient_funds',
      }),
    ]);
    await fixture.whenStable();
    fixture.detectChanges();
    const btn = el.querySelector<HTMLButtonElement>('[data-territory="city"] .enter');
    expect(btn?.disabled).toBeTrue();
    expect(el.querySelector('[data-territory="city"]')?.textContent).toContain('$4,000 more');
  });

  it('asks before entering, then enters', async () => {
    const { api, fixture, el } = setup(
      [
        territory(),
        territory({ key: 'city', name: 'City', entered: false, entryCost: 500, enterBlocked: '' }),
      ],
      1000,
    );
    await fixture.whenStable();
    fixture.detectChanges();
    el.querySelector<HTMLButtonElement>('[data-territory="city"] .enter')!.click();
    fixture.detectChanges();
    expect(api.enterTerritory).not.toHaveBeenCalled();
    el.querySelector<HTMLButtonElement>(
      'app-confirm-dialog .btn-primary, app-confirm-dialog button:last-of-type',
    )!.click();
    await fixture.whenStable();
    expect(api.enterTerritory).toHaveBeenCalledWith('city');
  });
});
