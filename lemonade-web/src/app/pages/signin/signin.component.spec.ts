import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';

import { FakeAuthPort, provideFakeAuth } from '../../core/testing/fake-auth';
import { SigninComponent } from './signin.component';

describe('SigninComponent', () => {
  let fixture: ComponentFixture<SigninComponent>;
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
    navigate = spyOn(TestBed.inject(Router), 'navigateByUrl').and.resolveTo(true);
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(SigninComponent);
    fixture.detectChanges();
    el = fixture.nativeElement;
  });

  afterEach(() => http.verify());

  const button = () => el.querySelector<HTMLButtonElement>('button')!;
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    fixture.detectChanges();
  };

  it('has exactly one button, the primary "Continue with Google"', () => {
    expect(el.querySelectorAll('button').length).toBe(1);
    expect(button().textContent!.trim()).toBe('Continue with Google');
    expect(button().classList).toContain('btn-primary');
    expect(el.querySelector('input')).toBeNull();
  });

  it('signs in and goes to the game when the account has a profile', async () => {
    button().click();
    await settle();
    http.expectOne('/api/me').flush({ id: 1, username: 'lemonjoe' });
    await settle();

    expect(port.popupCalls).toBe(1);
    expect(navigate).toHaveBeenCalledWith('/game');
  });

  it('sends a new account to choose a username', async () => {
    button().click();
    await settle();
    http
      .expectOne('/api/me')
      .flush({ error: 'profile_required', message: 'x' }, { status: 403, statusText: '' });
    await settle();

    expect(navigate).toHaveBeenCalledWith('/signin/username');
  });

  it('shows an inline error when Google sign-in fails', async () => {
    port.popupError = { code: 'auth/network-request-failed' };
    button().click();
    await settle();

    expect(el.querySelector('.field-error')?.textContent).toContain('Google sign-in did not work');
    expect(navigate).not.toHaveBeenCalled();
    expect(button().disabled).toBeFalse();
  });

  it('stays put when the popup is closed', async () => {
    port.popupError = { code: 'auth/popup-closed-by-user' };
    button().click();
    await settle();

    expect(el.querySelector('.field-error')).toBeNull();
    expect(navigate).not.toHaveBeenCalled();
  });

  it('disables the button while offline', () => {
    window.dispatchEvent(new Event('offline'));
    fixture.detectChanges();
    expect(button().disabled).toBeTrue();
    window.dispatchEvent(new Event('online'));
  });
});
