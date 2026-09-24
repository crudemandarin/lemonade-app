// Builds design-preview/event-backdrops.html: a standalone page (no Angular) that
// renders the real event-backdrop SCSS so the scenes can be reviewed on their own.
// Usage: node design-preview/build.mjs   (from lemonade-web/), then open the HTML file.
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import * as sass from 'sass';

const here = (p) => fileURLToPath(new URL(p, import.meta.url));
const css = sass.compile(here('../src/app/shared/event-backdrop/event-backdrop.component.scss')).css;
const tokens = readFileSync(here('../src/styles.scss'), 'utf8');
const themeCss = sass.compileString(tokens).css;
const page = readFileSync(here('template.html'), 'utf8')
  .replace('/*THEME*/', themeCss)
  .replace('/*BACKDROP*/', css);
writeFileSync(here('event-backdrops.html'), page);
console.log('wrote design-preview/event-backdrops.html');
