import { TestBed } from '@angular/core/testing';

import { HELP_SECTIONS, HELP_TERMS } from './glossary';
import { HelpPanelComponent } from './help-panel.component';
import { HelpService } from './help.service';

describe('glossary', () => {
  it('has unique ids and known sections', () => {
    const ids = HELP_TERMS.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    const sections = HELP_SECTIONS.map((s) => s.id);
    for (const t of HELP_TERMS) expect(sections).toContain(t.section);
  });

  it('renders every term in its section', () => {
    const fixture = TestBed.createComponent(HelpPanelComponent);
    const help = TestBed.inject(HelpService);
    help.open.set(true);
    for (const s of HELP_SECTIONS) {
      help.section.set(s.id);
      fixture.detectChanges();
      const el: HTMLElement = fixture.nativeElement;
      for (const t of HELP_TERMS.filter((x) => x.section === s.id)) {
        expect(el.querySelector('#help-' + t.id))
          .withContext(t.id)
          .not.toBeNull();
      }
    }
  });

  it('show() opens the panel on the right tab', () => {
    const help = TestBed.inject(HelpService);
    help.show('ice');
    expect(help.open()).toBeTrue();
    expect(help.section()).toBe('resources');
    expect(help.target()).toBe('ice');
  });
});

describe('HelpLinkComponent', () => {
  it('opens the panel at its term', async () => {
    const { HelpLinkComponent } = await import('./help-link.component');
    const fixture = TestBed.createComponent(HelpLinkComponent);
    fixture.componentRef.setInput('term', 'bid');
    fixture.componentRef.setInput('label', 'the bid');
    fixture.detectChanges();
    const button: HTMLButtonElement = fixture.nativeElement.querySelector('button');
    expect(button.getAttribute('aria-label')).toBe('What is the bid?');
    button.click();
    const help = TestBed.inject(HelpService);
    expect(help.section()).toBe('market');
    expect(help.open()).toBeTrue();
  });
});
