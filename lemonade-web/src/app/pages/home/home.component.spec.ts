import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { SessionService } from '../../core/session.service';
import { HomeComponent } from './home.component';

describe('HomeComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
  });

  afterEach(() => TestBed.inject(SessionService).signOut());

  function render(): HTMLElement {
    const fixture = TestBed.createComponent(HomeComponent);
    fixture.detectChanges();
    return fixture.nativeElement;
  }

  function links(el: HTMLElement): [string, string | null][] {
    return Array.from(el.querySelectorAll('a')).map((a) => [
      a.textContent!.trim(),
      a.getAttribute('href'),
    ]);
  }

  it('offers Play game and Sign in when signed out', () => {
    TestBed.inject(SessionService).signOut();
    expect(links(render())).toEqual([
      ['Play game', '/signin'],
      ['Sign in', '/signin'],
    ]);
  });

  it('offers Continue game when signed in', () => {
    TestBed.inject(SessionService).signIn('lemonjoe');
    expect(links(render())).toEqual([['Continue game', '/game']]);
  });
});
