import { ComponentFixture, TestBed } from '@angular/core/testing';

import { VictoryBannerComponent } from './victory-banner.component';

describe('VictoryBannerComponent', () => {
  let fixture: ComponentFixture<VictoryBannerComponent>;
  let el: HTMLElement;

  function render(wonOnDay: number, runId = 'run-a'): void {
    fixture = TestBed.createComponent(VictoryBannerComponent);
    fixture.componentRef.setInput('runId', runId);
    fixture.componentRef.setInput('wonOnDay', wonOnDay);
    fixture.componentRef.setInput('wonNetWorth', 21_500_000);
    fixture.detectChanges();
    el = fixture.nativeElement as HTMLElement;
  }

  beforeEach(() => {
    localStorage.removeItem('lemonade.victoryDismissed');
    TestBed.configureTestingModule({ imports: [VictoryBannerComponent] });
  });

  it('says nothing before a win', () => {
    render(0);
    expect(el.textContent?.trim()).toBe('');
  });

  it('names the day and the net worth, and says the run goes on', () => {
    render(152);
    expect(el.textContent).toContain('Global leader on day 152');
    expect(el.textContent).toContain('$21.5M');
    expect(el.textContent).toContain('Keep playing');
  });

  it('stays dismissed for that run only', () => {
    render(152);
    el.querySelector<HTMLButtonElement>('.dismiss')!.click();
    fixture.detectChanges();
    expect(el.querySelector('.victory')).toBeNull();

    render(152);
    expect(el.querySelector('.victory')).toBeNull();
    render(152, 'run-b');
    expect(el.querySelector('.victory')).not.toBeNull();
  });
});
