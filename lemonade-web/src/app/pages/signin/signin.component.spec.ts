import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed, fakeAsync, flushMicrotasks } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { FirebaseConfigService } from '../../core/firebase-config';
import { SessionService, USERNAME_KEY } from '../../core/session.service';
import { FakeAuthPort, provideFakeAuth } from '../../core/testing/fake-auth';
import { SigninComponent } from './signin.component';

describe('SigninComponent', () => {
  let fixture: ComponentFixture<SigninComponent>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let port: FakeAuthPort;
  let navigate: jasmine.Spy;

  function create(googleConfigured = true, url = '/signin') {
    port = new FakeAuthPort();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        provideFakeAuth(port),
      ],
    });
    if (googleConfigured) {
      TestBed.inject(FirebaseConfigService).config = {
        apiKey: 'k',
        authDomain: 'a',
        projectId: 'p',
        appId: 'i',
      };
    }
    const router = TestBed.inject(Router);
    navigate = spyOn(router, 'navigateByUrl').and.resolveTo(true);
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(SigninComponent);
    fixture.detectChanges();
    el = fixture.nativeElement;
    void url;
  }

  afterEach(() => {
    http.verify();
    TestBed.inject(SessionService).signOut();
    localStorage.removeItem(USERNAME_KEY);
  });

  const usernameInput = () => el.querySelector<HTMLInputElement>('input')!;
  const googleButton = () => el.querySelector<HTMLButtonElement>('button[type=button]')!;
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };

  function submit(username: string) {
    usernameInput().value = username;
    usernameInput().dispatchEvent(new Event('input'));
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    fixture.detectChanges();
  }

  describe('username', () => {
    beforeEach(() => create());

    it('has one primary button, "Continue", and Google is the quieter option', () => {
      expect(el.querySelectorAll('.btn-primary').length).toBe(1);
      expect(el.querySelector('.btn-primary')!.textContent!.trim()).toBe('Continue');
      expect(googleButton().textContent!.trim()).toBe('Sign in with Google');
      expect(googleButton().classList).not.toContain('btn-primary');
    });

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

    it('logs in on a username alone and routes to /game', fakeAsync(() => {
      submit(' LemonJoe ');
      const req = http.expectOne('/api/login');
      expect(req.request.body).toEqual({ username: 'lemonjoe' });
      req.flush({ id: 1, username: 'lemonjoe' });
      flushMicrotasks();

      expect(navigate).toHaveBeenCalledWith('/game');
    }));

    it('accepts a 3-character username', fakeAsync(() => {
      submit('Joe');
      const req = http.expectOne('/api/login');
      expect(req.request.body).toEqual({ username: 'joe' });
      req.flush({ id: 1, username: 'joe' });
      flushMicrotasks();
    }));

    it('says a protected username needs Google, and stays put', fakeAsync(() => {
      submit('lemonjoe');
      http.expectOne('/api/login').flush(
        {
          error: 'account_secured',
          message: 'This username is protected. Sign in with Google to play it.',
        },
        { status: 409, statusText: '' },
      );
      flushMicrotasks();
      fixture.detectChanges();

      expect(el.querySelector('.field-error')?.textContent).toContain('protected');
      expect(navigate).not.toHaveBeenCalled();
    }));

    it('disables Continue while offline', () => {
      window.dispatchEvent(new Event('offline'));
      fixture.detectChanges();
      expect(el.querySelector<HTMLButtonElement>('button[type=submit]')!.disabled).toBeTrue();
      window.dispatchEvent(new Event('online'));
    });
  });

  describe('Google', () => {
    beforeEach(() => create());

    it('signs in and goes to the game when the account has a profile', async () => {
      googleButton().click();
      await settle();
      http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
      await settle();

      expect(port.popupCalls).toBe(1);
      expect(navigate).toHaveBeenCalledWith('/game');
    });

    it('sends a new Google account to choose a username', async () => {
      googleButton().click();
      await settle();
      http
        .expectOne('/api/me')
        .flush({ error: 'profile_required', message: 'x' }, { status: 403, statusText: '' });
      await settle();

      expect(navigate).toHaveBeenCalledWith('/signin/username');
    });

    it('shows an inline error when Google sign-in fails', async () => {
      port.popupError = { code: 'auth/network-request-failed' };
      googleButton().click();
      await settle();

      expect(el.querySelector('.field-error')?.textContent).toContain(
        'Google sign-in did not work',
      );
      expect(navigate).not.toHaveBeenCalled();
    });

    it('stays put when the popup is closed', async () => {
      port.popupError = { code: 'auth/popup-closed-by-user' };
      googleButton().click();
      await settle();

      expect(el.querySelector('.field-error')).toBeNull();
      expect(navigate).not.toHaveBeenCalled();
    });
  });

  it('hides the Google option when the deployment has no Firebase settings', () => {
    create(false);
    expect(googleButton()).toBeNull();
    expect(el.querySelector('input')).not.toBeNull();
  });
});
