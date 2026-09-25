import { Pipe, PipeTransform } from '@angular/core';

/** Whole dollars: 1240 → "$1,240", -15 → "-$15". The domain has no cents. */
export function formatMoney(value: number): string {
  const formatted = `$${Math.abs(value).toLocaleString('en-US')}`;
  return value < 0 ? `-${formatted}` : formatted;
}

/**
 * Like formatMoney, but from a million up it is compact: 1234567 → "$1.2M", 6000000 → "$6M".
 * Show the exact value in a `title` wherever this is used for a value that can be that big.
 */
export function formatMoneyCompact(value: number): string {
  const abs = Math.abs(value);
  const units: [number, string][] = [
    [1_000_000_000, 'B'],
    [1_000_000, 'M'],
  ];
  for (const [size, suffix] of units) {
    if (abs >= size) {
      const scaled = Math.floor((abs / size) * 10) / 10;
      const text = `$${scaled.toFixed(1).replace(/\.0$/, '')}${suffix}`;
      return value < 0 ? `-${text}` : text;
    }
  }
  return formatMoney(value);
}

@Pipe({ name: 'money', standalone: true })
export class MoneyPipe implements PipeTransform {
  transform(value: number): string {
    return formatMoneyCompact(value);
  }
}
