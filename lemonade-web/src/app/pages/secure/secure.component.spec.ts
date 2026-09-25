import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { LINKING_KEY } from '../../core/auth.service';
import { SessionService, USERNAME_KEY } from '../../core/session.service';
import { FakeAuthPort, provideFakeAuth } from '../../core/testing/fake-auth';
import { SecureComponent } from './secure.component';

describe('SecureComponent', () => {
  let fixture: ComponentFixture<SecureComponent>;
  let el: HTMLElement;
  let http: HttpTestingController;
  let port: FakeAuthPort;
  let navigate: jasmine.Spy;

  beforeEach(() => {
    port = new FakeAuthPort();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        provideFakeAuth(port),
      ],
    });
    TestBed.inject(SessionService).signIn('lemonjoe');
    navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(SecureComponent);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  afterEach(() => {
    http.verify();
    TestBed.inject(SessionService).signOut();
    localStorage.removeItem(USERNAME_KEY);
    sessionStorage.removeItem(LINKING_KEY);
  });

  const primary = () => el.querySelector<HTMLButtonElement>('.btn-primary')!;
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };

  it('explains the risk in terms of the username and has one primary button', () => {
    expect(el.textContent).toContain('anyone who types lemonjoe can play as you');
    expect(el.querySelectorAll('.btn-primary').length).toBe(1);
    expect(primary().textContent!.trim()).toBe('Link Google account');
    expect(el.querySelector('a[href="/game"]')!.textContent).toContain('Not now');
  });

  it('links the account and returns to the game', async () => {
    primary().click();
    await settle();
    http.expectOne('/api/me/claim').flush({ id: 1, username: 'lemonjoe' });
    await settle();

    expect(TestBed.inject(SessionService).secured()).toBeTrue();
    expect(navigate).toHaveBeenCalledWith('/game');
  });

  const failures: [string, string][] = [
    ['already_linked', 'already plays as another username'],
    ['already_claimed', 'Someone secured this username first'],
    ['rate_limited', 'Too many attempts'],
  ];
  for (const [code, text] of failures) {
    it(`shows ${code} and leaves the player a guest`, async () => {
      primary().click();
      await settle();
      http
        .expectOne('/api/me/claim')
        .flush({ error: code, message: 'x' }, { status: 409, statusText: '' });
      await settle();

      expect(el.querySelector('.field-error')?.textContent).toContain(text);
      expect(TestBed.inject(SessionService).secured()).toBeFalse();
      expect(navigate).not.toHaveBeenCalled();
      expect(primary().disabled).toBeFalse();
    });
  }

  it('shows a Google sign-in failure', async () => {
    port.popupError = { code: 'auth/network-request-failed' };
    primary().click();
    await settle();

    expect(el.querySelector('.field-error')?.textContent).toContain('did not work');
  });

  it('disables the button while offline', () => {
    window.dispatchEvent(new Event('offline'));
    fixture.detectChanges();
    expect(primary().disabled).toBeTrue();
    window.dispatchEvent(new Event('online'));
  });
});
