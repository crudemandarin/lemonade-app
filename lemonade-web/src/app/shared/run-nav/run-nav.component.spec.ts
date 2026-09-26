import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { RunNavComponent } from './run-nav.component';

describe('RunNavComponent', () => {
  let fixture: ComponentFixture<RunNavComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
    fixture = TestBed.createComponent(RunNavComponent);
    fixture.detectChanges();
  });

  it('links to the game and the run pages', () => {
    const links = Array.from(fixture.nativeElement.querySelectorAll('a')).map((a) =>
      (a as HTMLAnchorElement).getAttribute('href'),
    );
    expect(links).toEqual(['/game', '/production', '/empire', '/upgrades']);
  });
});
