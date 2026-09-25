import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, provideRouter } from '@angular/router';

import { dayReport, runDetail } from '../../core/testing/fixtures';
import { RunComponent } from './run.component';

describe('RunComponent', () => {
  let http: HttpTestingController;
  let fixture: ComponentFixture<RunComponent>;
  let el: HTMLElement;

  function render() {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { paramMap: new Map([['id', 'run-9']]) } },
        },
      ],
    });
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(RunComponent);
    el = fixture.nativeElement;
    fixture.detectChanges();
  }
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };
  const summaries = [1, 2, 3].map((day) => ({
    day,
    produced: 4,
    capitalBefore: 1000,
    capitalAfter: 970,
    newEvents: [],
    expiredEvents: [],
    bankrupt: false,
  }));

  afterEach(() => http.verify());

  it('shows the score, how it ended, the totals, all three charts and the day reports', async () => {
    render();
    http.expectOne('/api/runs/run-9').flush(
      runDetail({
        runId: 'run-9',
        score: 2100,
        netWorth: 2100,
        capital: 1500,
        days: 9,
        endedBy: 'gave_up',
        isBest: true,
        reports: summaries,
      }),
    );
    await settle();
    http.expectOne('/api/game/reports/3?runId=run-9').flush(dayReport({ day: 3 }));
    await settle();

    expect(el.querySelector('h1')!.textContent).toContain('You called it on day 9');
    expect(el.querySelector('.score .value')!.textContent).toContain('$2,100');
    expect(el.querySelector('.score .best')!.textContent).toContain('Personal best');
    const worth = Array.from(el.querySelectorAll('.worth > div')).map((r) =>
      [r.querySelector('dt')!.textContent, r.querySelector('dd')!.textContent].join(' '),
    );
    expect(worth).toEqual(['Cash $1,500', 'Stock and buildings $600']);

    expect(el.querySelector('app-run-stats')).not.toBeNull();
    expect(el.querySelector('app-timeline-charts .capital')).not.toBeNull();
    expect(el.querySelector('app-timeline-charts .stock')).not.toBeNull();
    expect(el.querySelector('app-timeline-charts .prices')).not.toBeNull();
    expect(el.querySelector('app-report-stepper h3')!.textContent).toContain('Day 3 report');
  });

  it('words a bankruptcy differently and hides the best badge when it is not the best', async () => {
    render();
    http
      .expectOne('/api/runs/run-9')
      .flush(runDetail({ endedBy: 'bankrupt', days: 5, isBest: false }));
    await settle();
    expect(el.querySelector('h1')!.textContent).toContain('Bankrupt on day 5');
    expect(el.querySelector('.score .best')).toBeNull();
  });

  it('steps through earlier days, fetching each once', async () => {
    render();
    http.expectOne('/api/runs/run-9').flush(runDetail({ runId: 'run-9', reports: summaries }));
    await settle();
    http.expectOne('/api/game/reports/3?runId=run-9').flush(dayReport({ day: 3 }));
    await settle();

    el.querySelector<HTMLButtonElement>('app-report-stepper .prev')!.click();
    await settle();
    http.expectOne('/api/game/reports/2?runId=run-9').flush(dayReport({ day: 2 }));
    await settle();
    expect(el.querySelector('app-report-stepper h3')!.textContent).toContain('Day 2 report');

    el.querySelector<HTMLButtonElement>('app-report-stepper .next')!.click();
    await settle();
    http.expectNone('/api/game/reports/3?runId=run-9');
    expect(el.querySelector('app-report-stepper h3')!.textContent).toContain('Day 3 report');
  });

  it('says so for a run that kept no day reports', async () => {
    render();
    http.expectOne('/api/runs/run-9').flush(runDetail({ reports: [] }));
    await settle();
    expect(el.querySelector('app-report-stepper .empty')!.textContent).toContain(
      'No day reports were kept',
    );
  });

  it('shows a not-found message for a run that is not yours', async () => {
    render();
    http
      .expectOne('/api/runs/run-9')
      .flush({ error: 'not_found' }, { status: 404, statusText: '' });
    await settle();
    expect(el.querySelector('.missing')!.textContent).toContain("wasn't found");
    expect(el.querySelector('app-timeline-charts')).toBeNull();
  });

  it('shows an error with a retry for other failures', async () => {
    render();
    http.expectOne('/api/runs/run-9').flush({}, { status: 500, statusText: '' });
    await settle();
    expect(el.querySelector('[role=alert]')!.textContent).toContain("Couldn't load this run");
    el.querySelector<HTMLButtonElement>('.retry')!.click();
    http.expectOne('/api/runs/run-9').flush(runDetail());
    await settle();
    expect(el.querySelector('h1')).not.toBeNull();
  });
});
