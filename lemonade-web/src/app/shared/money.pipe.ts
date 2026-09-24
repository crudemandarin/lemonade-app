import { Pipe, PipeTransform } from '@angular/core';

/** Whole dollars: 1240 → "$1,240", -15 → "-$15". The domain has no cents. */
export function formatMoney(value: number): string {
  const formatted = `$${Math.abs(value).toLocaleString('en-US')}`;
  return value < 0 ? `-${formatted}` : formatted;
}

@Pipe({ name: 'money', standalone: true })
export class MoneyPipe implements PipeTransform {
  transform(value: number): string {
    return formatMoney(value);
  }
}
