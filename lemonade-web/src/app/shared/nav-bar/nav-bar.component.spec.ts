import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { NavBarComponent } from './nav-bar.component';

describe('NavBarComponent', () => {
  let fixture: ComponentFixture<NavBarComponent>;
  let el: HTMLElement;

  function render(username: string | null) {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
    fixture = TestBed.createComponent(NavBarComponent);
    fixture.componentRef.setInput('username', username);
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  it('shows Sign in when signed out', () => {
    render(null);
    expect(el.textContent).toContain('Sign in');
    expect(el.textContent).not.toContain('Log out');
  });

  it('shows the username and emits logOut when signed in', () => {
    render('lemonjoe');
    let count = 0;
    fixture.componentInstance.logOut.subscribe(() => count++);

    expect(el.textContent).toContain('lemonjoe');
    el.querySelector<HTMLButtonElement>('.log-out')!.click();

    expect(count).toBe(1);
  });

  it('links to the scores page when signed in', () => {
    render('lemonjoe');
    expect(el.querySelector<HTMLAnchorElement>('a.scores-link')!.getAttribute('href')).toBe(
      '/scores',
    );
  });

  it('links to the upgrades page when signed in, and not when signed out', () => {
    render('lemonjoe');
    expect(el.querySelector<HTMLAnchorElement>('a.upgrades-link')!.getAttribute('href')).toBe(
      '/upgrades',
    );
    TestBed.resetTestingModule();
    render(null);
    expect(el.querySelector('a.upgrades-link')).toBeNull();
  });

  it('has no scores link when signed out', () => {
    render(null);
    expect(el.querySelector('a.scores-link')).toBeNull();
  });
});
