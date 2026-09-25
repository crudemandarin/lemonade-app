import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed, fakeAsync, flushMicrotasks } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { SessionService } from '../../core/session.service';
import { SigninComponent } from './signin.component';

describe('SigninComponent', () => {
  let fixture: ComponentFixture<SigninComponent>;
  let el: HTMLElement;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    fixture = TestBed.createComponent(SigninComponent);
    http = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  afterEach(() => {
    http.verify();
    TestBed.inject(SessionService).signOut();
  });

  function submit(username: string) {
    const input = el.querySelector<HTMLInputElement>('input')!;
    input.value = username;
    input.dispatchEvent(new Event('input'));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    fixture.detectChanges();
  }

  it('shows an inline error for an empty username and sends nothing', () => {
    submit('   ');
    expect(el.querySelector('.field-error')?.textContent).toContain('Enter a username');
  });

  it('rejects names with spaces or non-ASCII characters without calling the API', () => {
    for (const bad of ['a bcde', 'josé!', 'ab']) {
      submit(bad);
      expect(el.querySelector('.field-error')?.textContent).toContain('3 to 40');
    }
  });

  it('logs in and routes to /game', fakeAsync(() => {
    const navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);

    submit(' LemonJoe ');
    const req = http.expectOne('/api/login');
    expect(req.request.body).toEqual({ username: 'lemonjoe' });
    req.flush({ id: 1, username: 'lemonjoe' });
    flushMicrotasks();

    expect(navigate).toHaveBeenCalledWith('/game');
  }));

  it('accepts a 3-character username', fakeAsync(() => {
    spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    submit('Joe');
    const req = http.expectOne('/api/login');
    expect(req.request.body).toEqual({ username: 'joe' });
    req.flush({ id: 1, username: 'joe' });
    flushMicrotasks();
  }));

  it('disables Continue while offline', () => {
    window.dispatchEvent(new Event('offline'));
    fixture.detectChanges();
    expect(el.querySelector<HTMLButtonElement>('button[type=submit]')!.disabled).toBeTrue();
    window.dispatchEvent(new Event('online'));
  });
});
