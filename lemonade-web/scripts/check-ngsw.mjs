// After `npm run build`: the service worker must not cache the Firebase runtime config
// (a redeploy to another GCP project could keep serving the old one), and must not
// answer Firebase's /__/auth/* sign-in pages with the app shell.
import { readFileSync } from 'node:fs';

const ngsw = JSON.parse(readFileSync('dist/lemonade-web/browser/ngsw.json', 'utf8'));
const problems = [];

const cached = ngsw.assetGroups.flatMap((group) => group.urls);
for (const url of cached.filter((u) => u.startsWith('/config/'))) {
  problems.push(`ngsw.json caches ${url}`);
}
if ((ngsw.dataGroups ?? []).length > 0) {
  problems.push('dataGroups are defined: they could match Google or Firebase hosts');
}

const isNavigation = (path) => {
  const rules = ngsw.navigationUrls.map((u) => ({ ok: u.positive, re: new RegExp(u.regex) }));
  return rules.some((r) => r.ok && r.re.test(path)) && !rules.some((r) => !r.ok && r.re.test(path));
};
for (const path of ['/__/auth/handler', '/__/firebase/init.json', '/config/firebase-config.json']) {
  if (isNavigation(path)) {
    problems.push(`${path} is a service worker navigation URL`);
  }
}

if (problems.length > 0) {
  console.error(problems.join('\n'));
  process.exit(1);
}
console.log('ngsw.json ok: /config/** is not cached and /__/* bypasses the service worker');
