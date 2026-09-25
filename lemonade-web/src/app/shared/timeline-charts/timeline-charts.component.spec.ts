import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PricePoint, TimelinePoint } from '../../core/api.models';
import { timelinePoint } from '../../core/testing/fixtures';
import { TimelineChartsComponent } from './timeline-charts.component';

const POINTS: TimelinePoint[] = [
  timelinePoint({ kind: 'start', day: 1, capital: 1000 }),
  timelinePoint({
    kind: 'buy',
    day: 1,
    resource: 'lemon',
    qty: 3,
    amount: 66,
    capital: 934,
    stock: [3, 0, 0, 0, 0],
  }),
  timelinePoint({
    kind: 'sell',
    day: 1,
    resource: 'lemon',
    qty: 1,
    amount: 18,
    capital: 952,
    stock: [2, 0, 0, 0, 0],
  }),
  timelinePoint({
    kind: 'expand',
    day: 1,
    facility: 'warehouse',
    resource: 'ice',
    qty: 1,
    amount: 100,
    capital: 852,
    stock: [2, 0, 0, 0, 0],
  }),
  timelinePoint({
    kind: 'upgrade',
    day: 1,
    facility: 'production',
    amount: 1000,
    capital: 0,
    stock: [2, 0, 0, 0, 0],
  }),
  timelinePoint({
    kind: 'end_day',
    day: 1,
    produced: 0,
    amount: 30,
    capital: 822,
    stock: [2, 0, 0, 0, 0],
  }),
];

describe('TimelineChartsComponent', () => {
  let fixture: ComponentFixture<TimelineChartsComponent>;
  let el: HTMLElement;

  async function render(points: TimelinePoint[] = POINTS, priceLog: PricePoint[] = []) {
    fixture = TestBed.createComponent(TimelineChartsComponent);
    fixture.nativeElement.style.display = 'block';
    fixture.nativeElement.style.width = '640px';
    fixture.componentRef.setInput('points', points);
    fixture.componentRef.setInput('priceLog', priceLog);
    fixture.componentRef.setInput('basePrices', [20, 10, 10, 10, 90]);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  const click = (selector: string) => {
    el.querySelector<HTMLElement>(selector)!.click();
    fixture.detectChanges();
  };

  /** Moves the pointer over a chart's hit area, at a fraction of its width. */
  function hover(chart: 'capital' | 'stock' | 'prices', fraction: number) {
    const hit = el.querySelector<SVGRectElement>(`.${chart} .hit`)!;
    const box = hit.getBoundingClientRect();
    hit.dispatchEvent(
      new PointerEvent('pointermove', {
        clientX: box.left + box.width * fraction,
        clientY: box.top + 5,
      }),
    );
    fixture.detectChanges();
  }

  const markers = () => Array.from(el.querySelectorAll<SVGElement>('.capital .marker'));

  it('draws a capital chart and a stock chart', async () => {
    await render();
    expect(el.querySelector('.capital svg')).not.toBeNull();
    expect(el.querySelector('.stock svg')).not.toBeNull();
    expect(el.querySelector('.capital .capital-line')).not.toBeNull();
  });

  it('puts a dot on the capital line for every trade and facility purchase', async () => {
    await render();
    expect(markers().map((m) => m.getAttribute('data-kind'))).toEqual([
      'buy',
      'sell',
      'expand',
      'upgrade',
    ]);
  });

  it('draws no dot for the start or the end of a day', async () => {
    await render([POINTS[0], POINTS[5]]);
    expect(markers().length).toBe(0);
  });

  it('labels the latest capital at the end of the line', async () => {
    await render();
    expect(el.querySelector('.capital .end-label')!.textContent).toContain('$822');
  });

  it('shows one line per resource with a legend entry for each', async () => {
    await render();
    expect(el.querySelectorAll('.stock .stock-line').length).toBe(5);
    const legend = Array.from(el.querySelectorAll('.stock .legend button')).map((b) =>
      b.textContent!.trim(),
    );
    expect(legend).toEqual(['Lemon', 'Sugar', 'Ice', 'Cups', 'Lemonade']);
  });

  it('hides and shows a resource from the legend, but never hides them all', async () => {
    await render();
    const button = (name: string) =>
      Array.from(el.querySelectorAll<HTMLButtonElement>('.stock .legend button')).find((b) =>
        b.textContent!.includes(name),
      )!;

    button('Lemon').click();
    fixture.detectChanges();
    expect(el.querySelector('.stock-line[data-resource=lemon]')).toBeNull();
    expect(button('Lemon').getAttribute('aria-pressed')).toBe('false');

    button('Lemon').click();
    fixture.detectChanges();
    expect(el.querySelector('.stock-line[data-resource=lemon]')).not.toBeNull();

    for (const name of ['Lemon', 'Sugar', 'Ice', 'Cups']) {
      button(name).click();
      fixture.detectChanges();
    }
    button('Lemonade').click(); // would hide the last one
    fixture.detectChanges();
    expect(el.querySelectorAll('.stock .stock-line').length).toBe(1);
  });

  it('keeps each resource color when another is hidden', async () => {
    await render();
    const before = el.querySelector('.stock-line[data-resource=ice]')!.getAttribute('data-color');
    el.querySelector<HTMLButtonElement>('.stock .legend button')!.click(); // hide lemon
    fixture.detectChanges();
    expect(el.querySelector('.stock-line[data-resource=ice]')!.getAttribute('data-color')).toBe(
      before,
    );
  });

  it('shows a tooltip and a crosshair on both charts when hovering one', async () => {
    await render();
    hover('capital', 0);
    const tip = el.querySelector('.capital .tip')!;
    expect(tip.textContent).toContain('$1,000');
    expect(tip.textContent).toContain('Game started');
    expect(el.querySelector('.capital .cross')).not.toBeNull();
    expect(el.querySelector('.stock .cross')).not.toBeNull();
    expect(el.querySelector('.stock .tip')).toBeNull(); // only the hovered chart shows the readout
  });

  it('describes the action under the pointer', async () => {
    await render();
    hover('capital', 1); // the far right is the end of the day
    expect(el.querySelector('.capital .tip')!.textContent).toContain('Day 1 ended');
  });

  it('lists every resource in the stock tooltip', async () => {
    await render();
    hover('stock', 1);
    const rows = Array.from(el.querySelectorAll('.stock .tip li')).map((li) =>
      li.textContent!.replace(/\s+/g, ' ').trim(),
    );
    expect(rows.length).toBe(5);
    expect(rows[0]).toContain('2');
    expect(rows[0]).toContain('Lemon');
  });

  it('clears the hover when the pointer leaves', async () => {
    await render();
    hover('capital', 0.5);
    el.querySelector('.capital .hit')!.dispatchEvent(new PointerEvent('pointerleave'));
    fixture.detectChanges();
    expect(el.querySelector('.tip')).toBeNull();
    expect(el.querySelector('.cross')).toBeNull();
  });

  it('can be read with the keyboard', async () => {
    await render();
    const region = el.querySelector<HTMLElement>('.chart')!;
    expect(region.getAttribute('tabindex')).toBe('0');

    region.dispatchEvent(new KeyboardEvent('keydown', { key: 'End' }));
    fixture.detectChanges();
    expect(el.querySelector('.capital .tip')!.textContent).toContain('Day 1 ended');

    region.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft' }));
    fixture.detectChanges();
    expect(el.querySelector('.capital .tip')!.textContent).toContain('Upgraded production');

    region.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    fixture.detectChanges();
    expect(el.querySelector('.tip')).toBeNull();
  });

  it('offers the same data as a table', async () => {
    await render();
    click('.table-toggle');
    expect(el.querySelector('.chart')).toBeNull();
    const rows = el.querySelectorAll('table tbody tr');
    expect(rows.length).toBe(POINTS.length);
    expect(rows[1].textContent).toContain('Bought 3 lemon for $66');
    expect(rows[1].textContent).toContain('$934');
    click('.table-toggle');
    expect(el.querySelector('.chart')).not.toBeNull();
  });

  it('says so when there is not enough history to draw yet', async () => {
    await render([POINTS[0]]);
    expect(el.textContent).toContain('Your history will appear here');
  });

  it('explains the marker shapes', async () => {
    await render();
    const text = el.querySelector('.marker-legend')!.textContent!;
    for (const word of ['Bought', 'Sold', 'Built', 'Upgraded']) {
      expect(text).toContain(word);
    }
  });

  it('only ever labels the stock axis with whole cases', async () => {
    await render([POINTS[0], POINTS[5]].map((p) => ({ ...p, stock: [0, 0, 0, 0, 0] })));
    const labels = Array.from(el.querySelectorAll('.stock .tick'))
      .map((t) => t.textContent!.trim())
      .filter(
        (t) => /^[\d.]+$/.test(t) && !el.querySelector(`.stock .axis-title`)?.isEqualNode(null),
      );
    const yLabels = Array.from(el.querySelectorAll('.stock svg text.tick[text-anchor=end]')).map(
      (t) => t.textContent!.trim(),
    );
    expect(yLabels.length).toBeGreaterThan(1);
    expect(labels.length).toBeGreaterThan(0);
    for (const label of yLabels.filter((t) => t !== 'Day')) {
      expect(Number.isInteger(Number(label)))
        .withContext(label)
        .toBeTrue();
    }
  });

  describe('price chart', () => {
    const DAYS: TimelinePoint[] = [
      timelinePoint({ kind: 'start', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 1 }),
      timelinePoint({ kind: 'end_day', day: 2 }),
    ];
    const LOG: PricePoint[] = [
      { day: 1, prices: [20, 10, 10, 10, 90], events: [] },
      { day: 2, prices: [22, 9, 10, 11, 126], events: ['Heat Wave'] },
      { day: 3, prices: [21, 9, 10, 10, 100], events: [] },
    ];

    it('draws one line per commodity, coloured per resource', async () => {
      await render(DAYS, LOG);
      const lines = Array.from(el.querySelectorAll<SVGPathElement>('.prices .price-line'));
      expect(lines.map((l) => l.getAttribute('data-resource'))).toEqual([
        'lemon',
        'sugar',
        'ice',
        'cup',
        'lemonade',
      ]);
      expect(lines[4].style.stroke).toContain('--series-lemonade');
    });

    it('toggles a commodity from its legend but never hides them all', async () => {
      await render(DAYS, LOG);
      const buttons = () => Array.from(el.querySelectorAll<HTMLElement>('.prices .legend button'));
      buttons()[4].click();
      fixture.detectChanges();
      expect(el.querySelectorAll('.prices .price-line').length).toBe(4);
      for (const b of buttons().slice(0, 4)) {
        b.click();
        fixture.detectChanges();
      }
      expect(el.querySelectorAll('.prices .price-line').length).toBe(1);
    });

    it('switches between dollars and percent of base', async () => {
      await render(DAYS, LOG);
      const labels = () =>
        Array.from(el.querySelectorAll('.prices svg text.tick[text-anchor=end]')).map((t) =>
          t.textContent!.trim(),
        );
      expect(labels().some((l) => l.startsWith('$'))).toBeTrue();
      click('.prices .mode button:last-child');
      expect(labels().some((l) => l.endsWith('%'))).toBeTrue();
      expect(
        el.querySelector('.prices .mode button:last-child')!.getAttribute('aria-pressed'),
      ).toBe('true');
    });

    it('shades event days and names them in the tooltip', async () => {
      await render(DAYS, LOG);
      expect(el.querySelectorAll('.prices .event-band').length).toBe(1);
      hover('prices', 0.5); // day 2
      const tip = el.querySelector('.prices .tip')!.textContent!.replace(/\s+/g, ' ');
      expect(tip).toContain('Day 2: Heat Wave');
      expect(tip).toContain('$126');
      expect(el.querySelector('.capital .cross')).not.toBeNull(); // the crosshair is shared
    });

    it('shows percent of base in the tooltip', async () => {
      await render(DAYS, LOG);
      click('.prices .mode button:last-child');
      hover('prices', 0.5);
      expect(el.querySelector('.prices .tip')!.textContent).toContain('140%');
    });

    it('says so when there are not enough days yet', async () => {
      await render(DAYS, [LOG[0]]);
      expect(el.querySelector('.prices .empty')!.textContent).toContain('Prices will appear');
    });

    it('has a table twin with every day', async () => {
      await render(DAYS, LOG);
      click('.table-toggle');
      const rows = el.querySelectorAll('.price-table tbody tr');
      expect(rows.length).toBe(3);
      expect(rows[1].textContent).toContain('Heat Wave');
      expect(rows[1].textContent).toContain('$126');
    });
  });
});
