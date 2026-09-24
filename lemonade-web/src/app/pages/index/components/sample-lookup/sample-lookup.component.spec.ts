import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SampleLookupComponent } from './sample-lookup.component';

describe('SampleLookupComponent', () => {
  let fixture: ComponentFixture<SampleLookupComponent>;
  let httpMock: HttpTestingController;
  let el: HTMLElement;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [SampleLookupComponent],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    fixture = TestBed.createComponent(SampleLookupComponent);
    httpMock = TestBed.inject(HttpTestingController);
    el = fixture.nativeElement;
    fixture.detectChanges();
  });

  afterEach(() => httpMock.verify());

  function lookUp(id: string): void {
    const input = el.querySelector('input')!;
    input.value = id;
    input.dispatchEvent(new Event('input'));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    fixture.detectChanges();
  }

  for (const id of ['abc', '0', '1.5', '-3']) {
    it(`rejects "${id}" without calling the API`, () => {
      lookUp(id);

      httpMock.expectNone(() => true);
      expect(el.querySelector('.field-error')?.textContent).toContain(
        'ID must be a positive whole number.',
      );
    });
  }

  it('shows the sample returned by the API', () => {
    lookUp('7');

    httpMock.expectOne({ method: 'GET', url: '/api/samples/7' }).flush({
      id: 7,
      name: 'found me',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    });
    fixture.detectChanges();

    const result = el.querySelector('.result')?.textContent;
    expect(result).toContain('#7');
    expect(result).toContain('found me');
  });

  it('shows "sample not found" for a 404', () => {
    lookUp('999');

    httpMock
      .expectOne('/api/samples/999')
      .flush({ error: 'sample not found' }, { status: 404, statusText: 'Not Found' });
    fixture.detectChanges();

    expect(el.querySelector('.result')).toBeNull();
    expect(el.querySelector('.field-error')?.textContent).toContain('sample not found');
  });
});
