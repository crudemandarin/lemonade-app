import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';

import { Sample } from '../../../../models/sample.model';
import { SampleTableComponent } from './sample-table.component';

const SAMPLES: Sample[] = [
  { id: 2, name: 'world', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 1, name: 'hello', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
];

describe('SampleTableComponent', () => {
  let fixture: ComponentFixture<SampleTableComponent>;
  let httpMock: HttpTestingController;
  let el: HTMLElement;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [SampleTableComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideNoopAnimations()],
    });
    fixture = TestBed.createComponent(SampleTableComponent);
    httpMock = TestBed.inject(HttpTestingController);
    el = fixture.nativeElement;

    fixture.detectChanges(); // ngOnInit loads the list
    httpMock.expectOne({ method: 'GET', url: '/api/samples' }).flush(SAMPLES);
    fixture.detectChanges();
  });

  afterEach(() => httpMock.verify());

  const rows = () => Array.from(el.querySelectorAll<HTMLTableRowElement>('tbody tr'));
  const click = (row: HTMLElement, selector: string) => {
    row.querySelector<HTMLButtonElement>(selector)!.click();
    fixture.detectChanges();
  };

  it('loads samples on init and shows them sorted by ID', () => {
    expect(rows().map((r) => r.querySelector('.name')?.textContent?.trim())).toEqual([
      'hello',
      'world',
    ]);
    expect(el.querySelector('h2')?.textContent).toContain('Samples (2)');
  });

  it('edits a name inline and saves it with PUT', () => {
    click(rows()[0], 'button[title=Edit]');

    const input = rows()[0].querySelector('input')!;
    expect(input.value).toBe('hello');
    input.value = 'renamed';
    input.dispatchEvent(new Event('input'));
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    fixture.detectChanges();

    const req = httpMock.expectOne({ method: 'PUT', url: '/api/samples/1' });
    expect(req.request.body).toEqual({ name: 'renamed' });
    req.flush({ ...SAMPLES[1], name: 'renamed' });
    httpMock.expectOne('/api/samples').flush([SAMPLES[0], { ...SAMPLES[1], name: 'renamed' }]);
    fixture.detectChanges();

    expect(rows()[0].querySelector('input')).toBeNull();
    expect(rows()[0].querySelector('.name')?.textContent).toContain('renamed');
  });

  it('asks for confirmation before deleting', async () => {
    click(rows()[0], 'button[title=Delete]');

    expect(rows()[0].querySelector('.confirm')?.textContent).toContain('Delete?');
    httpMock.expectNone({ method: 'DELETE' });

    click(rows()[0], '.confirm-btn.yes');
    httpMock
      .expectOne({ method: 'DELETE', url: '/api/samples/1' })
      .flush(null, { status: 204, statusText: 'No Content' });
    httpMock.expectOne('/api/samples').flush([SAMPLES[0]]);
    fixture.detectChanges();
    await fixture.whenStable(); // let the row's :leave animation finish

    expect(rows().map((r) => r.querySelector('.name')?.textContent?.trim())).toEqual(['world']);
  });

  it('cancels the delete when No is clicked', () => {
    click(rows()[0], 'button[title=Delete]');
    click(rows()[0], '.confirm-btn:not(.yes)');

    expect(rows()[0].querySelector('.confirm')).toBeNull();
    httpMock.expectNone({ method: 'DELETE' });
  });
});
