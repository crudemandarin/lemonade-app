import { MoneyPipe, formatMoney, formatMoneyCompact } from './money.pipe';

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

  it('the pipe shows small values exactly', () => {
    expect(new MoneyPipe().transform(1240)).toBe('$1,240');
  });
});

describe('formatMoneyCompact', () => {
  it('leaves values under a million exact', () => {
    expect(formatMoneyCompact(999_999)).toBe('$999,999');
    expect(formatMoneyCompact(-1240)).toBe('-$1,240');
  });

  it('shortens millions and billions to one decimal, rounding down', () => {
    expect(formatMoneyCompact(1_000_000)).toBe('$1M');
    expect(formatMoneyCompact(1_234_567)).toBe('$1.2M');
    expect(formatMoneyCompact(6_000_000)).toBe('$6M');
    expect(formatMoneyCompact(-2_550_000)).toBe('-$2.5M');
    expect(formatMoneyCompact(3_400_000_000)).toBe('$3.4B');
  });

  it('is what the pipe shows', () => {
    expect(new MoneyPipe().transform(1_234_567)).toBe('$1.2M');
  });
});
