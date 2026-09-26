import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { EmpireStripComponent } from './empire-strip.component';

describe('EmpireStripComponent', () => {
  let fixture: ComponentFixture<EmpireStripComponent>;

  function render(goal: { kind: string; key: string; name: string; cost: number } | null): string {
    fixture.componentRef.setInput('era', 2);
    fixture.componentRef.setInput('eraName', 'City');
    fixture.componentRef.setInput('nextGoal', goal);
    fixture.detectChanges();
    return (fixture.nativeElement as HTMLElement).textContent ?? '';
  }

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [EmpireStripComponent],
      providers: [provideRouter([])],
    });
    fixture = TestBed.createComponent(EmpireStripComponent);
  });

  it('shows the era and the next goal with its price', () => {
    const text = render({ kind: 'territory', key: 'region', name: 'Region', cost: 50000 });
    expect(text).toContain('Era 2');
    expect(text).toContain('City');
    expect(text).toContain('Next: Region');
    expect(text).toContain('$50,000');
  });

  it('omits the goal when nothing is left', () => {
    expect(render(null)).not.toContain('Next:');
  });

  it('links to the Empire page', () => {
    render(null);
    const a = (fixture.nativeElement as HTMLElement).querySelector('a');
    expect(a?.getAttribute('href')).toBe('/empire');
  });
});
