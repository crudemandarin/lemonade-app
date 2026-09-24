import { MoneyPipe, formatMoney } from './money.pipe';

describe('formatMoney', () => {
  it('formats whole dollars with thousands separators', () => {
    expect(formatMoney(0)).toBe('$0');
    expect(formatMoney(15)).toBe('$15');
    expect(formatMoney(1240)).toBe('$1,240');
    expect(formatMoney(1234567)).toBe('$1,234,567');
  });

  it('puts the minus sign before the dollar sign', () => {
    expect(formatMoney(-15)).toBe('-$15');
    expect(formatMoney(-1240)).toBe('-$1,240');
  });

  it('the pipe delegates to it', () => {
    expect(new MoneyPipe().transform(1240)).toBe('$1,240');
  });
});
