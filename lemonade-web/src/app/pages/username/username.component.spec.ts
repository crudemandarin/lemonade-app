import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { AuthService } from '../../core/auth.service';
import { FakeAuthPort, provideFakeAuth } from '../../core/testing/fake-auth';
import { UsernameComponent } from './username.component';

describe('UsernameComponent', () => {
  let fixture: ComponentFixture<UsernameComponent>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let navigate: jasmine.Spy;

  beforeEach(() => {
    const port = new FakeAuthPort();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        provideFakeAuth(port),
      ],
    });
    navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    http = TestBed.inject(HttpTestingController);
    TestBed.inject(AuthService);
    fixture = TestBed.createComponent(UsernameComponent);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  afterEach(() => http.verify());

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };
  const primary = () => el.querySelector<HTMLButtonElement>('button.btn-primary')!;
  const error = () => el.querySelector('.field-error')?.textContent?.trim();

  function submit(username: string) {
    const input = el.querySelector<HTMLInputElement>('input')!;
    input.value = username;
    input.dispatchEvent(new Event('input'));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    fixture.detectChanges();
  }

  function toggle() {
    el.querySelector<HTMLButtonElement>('button.btn-ghost')!.click();
    fixture.detectChanges();
  }

  it('has one primary button and does not prefill the username', () => {
    expect(el.querySelectorAll('button.btn-primary').length).toBe(1);
    expect(primary().textContent!.trim()).toBe('Start playing');
    expect(el.querySelector('input')!.value).toBe('');
  });

  it('validates locally without calling the API', () => {
    submit('   ');
    expect(error()).toBe('Enter a username');
    for (const bad of ['a bcde', 'josé!', 'ab']) {
      submit(bad);
      expect(error()).toContain('3 to 40');
    }
  });

  it('creates a profile and goes to the game', async () => {
    submit(' LemonJoe ');
    const req = http.expectOne('/api/me/username');
    expect(req.request.body).toEqual({ username: 'lemonjoe' });
    req.flush({ id: 1, username: 'lemonjoe' });
    await settle();

    expect(navigate).toHaveBeenCalledWith('/game');
  });

  it('accepts a 3-character username', async () => {
    submit('Joe');
    http.expectOne('/api/me/username').flush({ id: 1, username: 'joe' });
    await settle();
  });

  const failures: [string, number, string, string][] = [
    ['username_taken', 409, 'That username is taken', 'create'],
    ['invalid_username', 400, '3 to 40', 'create'],
    ['unknown_username', 404, 'No player has that username', 'claim'],
    ['already_claimed', 409, 'belongs to another Google account', 'claim'],
    ['rate_limited', 429, 'Too many attempts', 'claim'],
  ];
  for (const [code, status, text, mode] of failures) {
    it(`shows ${code} inline`, async () => {
      if (mode === 'claim') {
        toggle();
      }
      submit('someone');
      http
        .expectOne(mode === 'claim' ? '/api/me/claim' : '/api/me/username')
        .flush({ error: code, message: 'server text' }, { status, statusText: '' });
      await settle();

      expect(error()).toContain(text);
      expect(navigate).not.toHaveBeenCalled();
      expect(primary().disabled).toBeFalse();
    });
  }

  describe('claiming an existing username', () => {
    beforeEach(toggle);

    it('switches the copy and explains what linking does, still with one primary button', () => {
      expect(el.querySelector('h1')!.textContent).toContain('Link your username');
      expect(primary().textContent!.trim()).toBe('Link username');
      expect(el.querySelectorAll('button.btn-primary').length).toBe(1);
      expect(el.textContent).toContain('links the old username');
    });

    it('claims and goes to the game', async () => {
      submit('OldTimer');
      const req = http.expectOne('/api/me/claim');
      expect(req.request.body).toEqual({ username: 'oldtimer' });
      req.flush({ id: 7, username: 'oldtimer' });
      await settle();

      expect(navigate).toHaveBeenCalledWith('/game');
    });

    it('can switch back and clears the error', () => {
      submit('x');
      expect(error()).toBeDefined();
      toggle();
      expect(el.querySelector('h1')!.textContent).toContain('Choose a username');
      expect(error()).toBeUndefined();
    });
  });

  it('carries on to the game if the account turns out to have a player already', async () => {
    submit('another');
    http
      .expectOne('/api/me/username')
      .flush({ error: 'already_linked', message: 'x' }, { status: 409, statusText: '' });
    await settle();
    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
    await settle();

    expect(navigate).toHaveBeenCalledWith('/game');
  });
});
