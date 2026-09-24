import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CreateSampleComponent } from './create-sample.component';

describe('CreateSampleComponent', () => {
  let fixture: ComponentFixture<CreateSampleComponent>;
  let httpMock: HttpTestingController;
  let el: HTMLElement;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CreateSampleComponent],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    fixture = TestBed.createComponent(CreateSampleComponent);
    httpMock = TestBed.inject(HttpTestingController);
    el = fixture.nativeElement;
    fixture.detectChanges();
  });

  afterEach(() => httpMock.verify());

  function submit(name: string): void {
    const input = el.querySelector('input')!;
    input.value = name;
    input.dispatchEvent(new Event('input'));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    fixture.detectChanges();
  }

  it('posts the trimmed name, reloads the list and clears the input', () => {
    submit('  hello  ');

    const req = httpMock.expectOne({ method: 'POST', url: '/api/samples' });
    expect(req.request.body).toEqual({ name: 'hello' });
    req.flush({ id: 1, name: 'hello', created_at: '', updated_at: '' });
    httpMock.expectOne({ method: 'GET', url: '/api/samples' }).flush([]);
    fixture.detectChanges();

    expect(el.querySelector('input')!.value).toBe('');
    expect(el.querySelector('.field-error')).toBeNull();
  });

  it("shows the API's error message when the request fails", () => {
    submit('');

    httpMock
      .expectOne({ method: 'POST', url: '/api/samples' })
      .flush({ error: 'name is required' }, { status: 400, statusText: 'Bad Request' });
    fixture.detectChanges();

    expect(el.querySelector('.field-error')?.textContent).toContain('name is required');
  });
});
