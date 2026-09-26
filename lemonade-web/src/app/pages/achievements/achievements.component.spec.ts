import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { achievement, achievementsResponse } from '../../core/testing/fixtures';
import { AchievementsComponent } from './achievements.component';

describe('AchievementsComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<AchievementsComponent>;
  let el: HTMLElement;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(AchievementsComponent);
    el = fixture.nativeElement;
    fixture.detectChanges();
  });

  afterEach(() => http.verify());

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };
  const text = () => el.textContent!.replace(/\s+/g, ' ');
  const row = (key: string) => el.querySelector(`[data-key=${key}]`)!;

  it('loads, groups by category and writes the tier and state as words', async () => {
    expect(text()).toContain('Loading achievements');
    http.expectOne('/api/achievements').flush(
      achievementsResponse({
        achievements: [
          achievement({
            key: 'nw_5k',
            category: 'wealth',
            tier: 'bronze',
            unlocked: true,
            unlockedAt: '2026-09-20T12:00:00Z',
          }),
          achievement({ key: 'nw_100k', name: 'Six figures', category: 'wealth', tier: 'silver' }),
        ],
      }),
    );
    await settle();

    expect(el.querySelector('app-card')!.textContent).toContain('Pocket money');
    expect(text()).toContain('1 of 2 unlocked');
    expect(row('nw_5k').textContent).toContain('bronze');
    expect(row('nw_5k').textContent).toContain('Unlocked');
    expect(row('nw_100k').textContent).toContain('silver');
    expect(row('nw_100k').textContent).toContain('Locked');
    // A category with no rows is not shown.
    expect(text()).not.toContain('Oddities');
  });

  it('shows a progress bar with its numbers', async () => {
    http.expectOne('/api/achievements').flush(
      achievementsResponse({
        achievements: [
          achievement({
            key: 'made_10k',
            name: 'Ten thousand cups',
            progress: { current: 6200, target: 10000 },
          }),
        ],
      }),
    );
    await settle();

    const bar = row('made_10k').querySelector('progress')!;
    expect(bar.value).toBe(6200);
    expect(bar.max).toBe(10000);
    expect(bar.getAttribute('aria-label')).toContain('Ten thousand cups');
    expect(row('made_10k').textContent).toContain('6200 of 10000');
  });

  it('shows a locked hidden achievement as ???, and its name once unlocked', async () => {
    http.expectOne('/api/achievements').flush(
      achievementsResponse({
        achievements: [
          achievement({
            key: 'just_ice',
            name: null,
            description: null,
            hidden: true,
            category: 'oddities',
          }),
          achievement({
            key: 'patience',
            name: 'Patience',
            hidden: true,
            unlocked: true,
            category: 'oddities',
            unlockedAt: '2026-09-20T12:00:00Z',
          }),
        ],
      }),
    );
    await settle();

    expect(row('just_ice').textContent).toContain('???');
    expect(row('just_ice').textContent).toContain('Hidden until you unlock it');
    expect(row('patience').textContent).toContain('Patience');
    expect(row('patience').textContent).not.toContain('???');
  });

  it('offers a retry when loading fails', async () => {
    http.expectOne('/api/achievements').flush({}, { status: 500, statusText: 'Server Error' });
    await settle();
    expect(el.querySelector('[role=alert]')).not.toBeNull();

    el.querySelector<HTMLButtonElement>('.retry')!.click();
    http.expectOne('/api/achievements').flush(achievementsResponse());
    await settle();
    expect(el.querySelector('[data-key=nw_5k]')).not.toBeNull();
  });
});
