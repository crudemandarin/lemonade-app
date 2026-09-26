import { ComponentFixture, TestBed } from '@angular/core/testing';

import { IconComponent } from './icon.component';

describe('IconComponent', () => {
  let fixture: ComponentFixture<IconComponent>;
  let el: HTMLElement;

  function render(name: string) {
    fixture = TestBed.createComponent(IconComponent);
    fixture.componentRef.setInput('name', name);
    fixture.detectChanges();
    el = fixture.nativeElement;
  }

  it('draws an assets/ui icon as a mask so it takes the text color', () => {
    render('trend-up');
    const mask = el.querySelector<HTMLElement>('.mask')!;
    expect(mask.style.getPropertyValue('--icon')).toContain('assets/ui/trend-up.svg');
  });

  it('draws icons without an asset inline', () => {
    render('x');
    expect(el.querySelector('svg path')).not.toBeNull();
  });

  it('draws catalog-keyed names that are not in the typed list', () => {
    render('upgrade-freezer_1');
    const mask = el.querySelector<HTMLElement>('.mask')!;
    expect(mask.style.getPropertyValue('--icon')).toContain('assets/ui/upgrade-freezer_1.svg');
  });
});
