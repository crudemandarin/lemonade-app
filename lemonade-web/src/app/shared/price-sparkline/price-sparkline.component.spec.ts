import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PriceSparklineComponent } from './price-sparkline.component';

describe('PriceSparklineComponent', () => {
  let fixture: ComponentFixture<PriceSparklineComponent>;
  let el: HTMLElement;

  function render(values: number[], label = 'Lemon') {
    fixture = TestBed.createComponent(PriceSparklineComponent);
    fixture.componentRef.setInput('values', values);
    fixture.componentRef.setInput('label', label);
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  const vertices = () =>
    (el.querySelector('path.line')?.getAttribute('d')?.match(/[ML]/g) ?? []).length;

  it('draws one vertex per day of history', () => {
    render([20, 22, 19, 23]);
    expect(vertices()).toBe(4);
  });

  it('marks the current price with an end dot', () => {
    render([20, 22, 23]);
    expect(el.querySelectorAll('circle.end').length).toBe(1);
  });

  it('draws just the dot on day 1, when there is no history yet', () => {
    render([20]);
    expect(el.querySelector('path.line')).toBeNull();
    expect(el.querySelectorAll('circle.end').length).toBe(1);
  });

  it('describes the trend for screen readers', () => {
    render([18, 20, 23], 'Lemon');
    const label = el.querySelector('svg')!.getAttribute('aria-label')!;
    expect(label).toContain('Lemon');
    expect(label).toContain('$18');
    expect(label).toContain('$23');
    expect(label).toContain('3 days');
  });

  it('keeps every point inside the drawing area, even when the price never moves', () => {
    render([10, 10, 10, 10]);
    const d = el.querySelector('path.line')!.getAttribute('d')!;
    const ys = [...d.matchAll(/[ML][\d.]+ ([\d.]+)/g)].map((m) => Number(m[1]));
    expect(ys.every((y) => Number.isFinite(y) && y > 0 && y < 28)).toBeTrue();
  });

  it('shows the price for the day under the pointer', () => {
    render([18, 20, 23]);
    const svg = el.querySelector('svg')!;
    const box = svg.getBoundingClientRect();
    svg.dispatchEvent(
      new PointerEvent('pointermove', { clientX: box.left + 1, clientY: box.top + 5 }),
    );
    fixture.detectChanges();
    expect(el.querySelector('.tip')?.textContent).toContain('$18');
    svg.dispatchEvent(new PointerEvent('pointerleave'));
    fixture.detectChanges();
    expect(el.querySelector('.tip')).toBeNull();
  });
});
