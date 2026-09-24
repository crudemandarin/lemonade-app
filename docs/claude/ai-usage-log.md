# AI usage log

Curated prompts and techniques that show deliberate, advanced use of AI tools. Not a transcript: only entries worth showing in the walkthrough.

## What qualifies
- Novel workflows (e.g. separated Designer / Implementor / Reviewer roles, file-based handoffs)
- Prompts that constrain or steer the model in a non-obvious way (test-first gating, scope guards, forced tradeoff analysis)
- Using AI to verify or challenge its own output (adversarial review, fresh-clone checks, spec-vs-code audits)
- Effective use of tooling features (plan mode, subagents, custom commands, hooks, context management)
- Cases where AI got it wrong and how the prompt or process was adjusted

## Entry template
### N. Short title
- **Phase:** Design / Build / Review / Polish
- **Prompt:** (verbatim, or trimmed with `[...]`)
- **Why it works:** what technique or insight is behind it
- **Result:** what it produced, and what changed as a result
- **Lesson:** what I'd reuse or avoid

## Entries

### 1. Docs as the spec for asset generation
- **Phase:** Build
- **Prompt:** "following the notes in docs/claude: create simple SVG icons for all resources, facility types (and their upgrade levels), a favicon and any other useful svg assets to be used by the app"
- **Why it works:** Pointing at the design docs instead of listing assets made the model derive the inventory itself: five resources, four warehouse tiers, four production tiers, and the PWA and UI icons DESIGN and SPEC imply. It also read the theme tokens, so the icons match the palette.
- **Result:** About 30 SVGs, a favicon, and regenerated PWA PNGs, wired into the manifest, `index.html`, and service worker config. The model rendered a contact sheet, found four facility icons that were invalid XML (duplicate attributes), and fixed them before reporting.
- **Lesson:** Ask for "all assets the docs imply" and let the model do the inventory. Have it render and look at its own output; a broken SVG only shows up visually.

### 2. Design pass before code, then a hard gate before integration
- **Phase:** Design / Build
- **Prompt:** "as per the design, there can be different RANDOM EVENTS (such as weather and holidays). review the possible effects and produce a list of ideas for background animations that we can pair with the random event", then (interrupting an early implementation) "produce a complete design for all these antimations, and then next we will impleemnt", then "begin implementation. do not integrate to frontend until after you build all the styles and i review first"
- **Why it works:** The first prompt grounds ideas in the real event table (the model checked DESIGN and the Go config for keys, multipliers, and durations). The interrupt forces a written design (`EVENT-BACKDROPS.md`: layering rules, particle budget, reduced motion, trade-offs, open questions) that I could answer in one line ("cap at two scenes, loop infinitely"). The last prompt puts a review gate between building and integrating.
- **Result:** A design doc, six scene stylesheets, a component with specs, and a standalone review page (`design-preview/`) that renders the real SCSS with event, theme, and reduced-motion toggles. Integration happened only after "looks great".
- **Lesson:** The model jumped straight to code twice, and the integration was already in a commit before the gate arrived, so it had to be removed by hand. State "design only" or "don't integrate" in the first prompt, not after the fact. Requiring a standalone preview makes visual review cheap.

### 3. Constraining a creative feature by its cost to the user
- **Phase:** Design
- **Prompt:** "I want these animations to play in a loop in the background of the page. it should fully cover the background dimensions of the page. becasue it'll be so large, the animations themselves shouldn't be too heavy on detail or distract from the foreground"
- **Why it works:** It states the constraint and the reason for it. The model turned "not too heavy" into concrete rules: CSS-only, about 40 elements per scene, compositor-only properties, 55% layer opacity, opaque cards on top, and a cap on simultaneous scenes.
- **Result:** Deterministic particles (golden-ratio spacing, negative delays so a scene starts mid-loop) and no `Math.random()`, which also made the component testable.
- **Lesson:** Give the reason for a constraint; the model then picks its own guardrails.

### 4. AI checks its own visual work with a real browser
- **Phase:** Review / Polish
- **Prompt:** "please intensify the following effects, they aren't very visible currently: [...]" and "update the frontend UI to be mobile compatible"
- **Why it works:** Both are visual, so the model drove headless Chrome: screenshots of each scene before and after tuning, then a Puppeteer script that intercepts `/api` calls with fixture data and injects the login into localStorage, so the real Angular app rendered at 320, 375, and 1100 px with no backend.
- **Result:** The mobile screenshot was 770 px wide for a 750 px viewport. That exposed a 10 px horizontal overflow in the market grid, which the script's overflow check had missed. The fix was `minmax(0, 1fr)` columns. A second screenshot pass also caught the upgrade button's upkeep note floating to the side.
- **Lesson:** Ask for evidence, not "it should work". Compare the screenshot's pixel width to the viewport; a cheap check for overflow.

### 5. A small UI request that needed a backend change
- **Phase:** Build
- **Prompt:** "for facilities section, it's not very readable. bold the lines about upkeep, facility level and type. on the upgrade level button, include the increase in upkeep cost (increase $x upkeep) for example" (plus "when your capital hits $0 but you still have enough inventory to survive, display a warning message on the day report")
- **Why it works:** The model traced each request to its data before coding. The upkeep increase wasn't in the API, so it added `upkeepIncrease` to the Go DTO with a test. The warning needed no API change, because `capitalAfter === 0 && !bankrupt` already implies stock remains, since bankruptcy requires both to be zero.
- **Result:** One commit spanning Go and Angular, with tests on both sides, and the values (`$10` and `$15`) checked against the tier config.
- **Lesson:** Ask for "the data path" of a UI change. Reusing an existing invariant avoided an unnecessary API field.

### 6. Frontend file as the API contract, and the model overrode the design doc with it
- **Phase:** Build
- **Prompt:** "follow docs (spec, design, claude, plan) in: docs/claude / refer to api.service.ts / your job is to implement the golang backend based on this spec."
- **Why it works:** Two sources of truth, one line each. The docs give the rules; the already-built client gives the wire contract. The model followed `api.service.ts` to `api.models.ts` (whose header says "the backend must return exactly these shapes") and to the facilities panel, instead of stopping at the docs.
- **Result:** It found that the frontend contradicts DESIGN: all warehouses share one level, and only building counts are per resource, so the routes take `warehouse`/`production` rather than six kinds. It built to the frontend and recorded the deviation as DECISIONS #7 and flagged it in the summary, rather than silently guessing (CLAUDE.md's "ask, don't guess" rule).
- **Lesson:** When a client already exists, point the model at it by name. Expect the docs to be stale in places, and require deviations to be written down.

### 7. Test-first caught an AI-written float bug; the AI's own tests were wrong too
- **Phase:** Build
- **Prompt:** (same session as #6; the CLAUDE.md workflow "write failing tests that encode the acceptance criteria, then implement" did the work)
- **Why it works:** The rounding table test (`price 100 -> bid 90 / ask 110`) came straight from SPEC rule 7. It failed with an ask of 111 because `100 * 1.1 = 110.00000000000001` and `ceil` rounded up. A second failure was the model's test under-funding itself (6 buildings x $200 > $1,000), so the test, not the code, was fixed.
- **Result:** A `snap` helper that rounds away float noise before floor/ceil (DECISIONS #9), plus a test that asserts bid < ask and bid >= 1 for every price from $1 to $500.
- **Lesson:** Specify examples from the spec, not from the implementation. When a test fails, decide which side is wrong before touching either.

### 8. Reviewing its own test for blast radius, then verifying against the real stack
- **Phase:** Review
- **Prompt:** (no user prompt; the model's own checks after the Postgres store was written)
- **Why it works:** The model noticed that its integration test ran `DELETE FROM games/users` against whatever `DATABASE_URL` pointed at, which could wipe a developer's database, and scoped the cleanup to `pguser%` rows before ever running it. It then ran the test against the live Docker Postgres (credentials sourced from `.env` without printing them), including 20 concurrent `Mutate` calls to prove the `FOR UPDATE` lock prevents lost updates, and finally started the server on a spare port and played a full day with curl.
- **Result:** A test that is safe to run on a shared DB, a lock behavior proven rather than assumed, and an end-to-end run showing the real JSON shapes and error bodies.
- **Lesson:** Ask "what does this test destroy if pointed at the wrong place?" before running it. Prefer one real-stack run over more mocks, and use a spare port so it doesn't collide with the running containers.

### 9. Tuning game balance by simulation, and refusing to trust the first scary number
- **Phase:** Build / Polish
- **Prompt:** "complete this readme part and pick out your own numbers and tuning based on what makes sense", then later "i think its too hard to lose the game. increase upkeep costs / other balance mechanisms"
- **Why it works:** "What makes sense" is untestable by feel, so the model built simulated players (a careful one, a sloppy one that forgets ice, a random one, an idle one) and ran hundreds of seeded games against the real domain code. Numbers replaced opinions: producing was unprofitable on only 1.4% of days, and 54-69% of careless players ended as "zombies" holding a little stock at $0 capital, so they could never lose. That showed the requested fix (raise upkeep) could not work alone, since even 5x upkeep changed nothing for careful play. The real cause was a rule (unpaid upkeep forgiven, any stock counts as grace), so the model proposed a rule change and asked for nothing more.
- **Result:** Lemonade $90, more volatile prices, doubled upkeep, and "upkeep is always owed; short cash sells stock, and if that is not enough the game ends". Outcomes went from "nobody loses" to careful about 1 in 8 bankrupt in 90 days, sloppy about 2 in 3 in 45 days, careless and idle always. The simulator became permanent `TestBalance*` guard-rail tests (bands, not exact numbers), so future tuning can't silently break the game feel.
- **Where the AI was wrong:** the first run under the new rule showed the "perfect" bot going bankrupt 12-52% of the time. Instead of tuning the game to that, the model inspected how it died: the bot reinvested to a cash reserve computed from the old upkeep, then upgraded and could not pay the new one. It fixed the bot (check affordability after the purchase) and reran, and only then chose the numbers.
- **Lesson:** Give the model a way to measure the thing it is asked to tune. When a result looks alarming, debug the instrument before changing the product. Ask for guard-rail tests so the tuning outlives the session.

### 10. Verifying "the backend matches the contract" by compiling real responses against it
- **Phase:** Review
- **Prompt:** "the backend is updated. please verify and connect the frontend to backend."
- **Why it works:** The other agent's DTO file said it "mirrors api.models.ts exactly", and the model did not take that on trust. It ran the real stack, played a full flow (login, buy, sell, both expands, upgrade, end-of-day with a live event, six error cases), turned every response into a TypeScript file assigning it to the frontend's own interfaces, and let `tsc --strict` check them. Missing, extra, or mistyped fields fail to compile.
- **Result:** The contract held, but the live run found two things unit tests on each side could not: the web proxy stripped the `/api` prefix the backend serves under (every call 404), and the backend charged $200/$500/$1,200 per warehouse upgrade where the addendum says $100/$250/$600, which left a new player at $0 after their first upgrade. A stale-username 401 also got a proper redirect to sign-in. A headless-Chrome run of the real UI against the real database then confirmed the whole loop.
- **Where the AI was wrong:** its own first script assumed a $500 upgrade was affordable after two expansions and failed on a correct 409. It said so, reordered the flow, and kept the failure as an error-shape check instead of forcing the assertion.
- **Lesson:** Use the type checker as the contract test: generate typed fixtures from the live system, not from either side's mocks. One real end-to-end run beats a page of mocks, and "one side says it matches" needs proof.

### 11. Asking the model's opinion on a design, then building it with a rules skill and screenshots
- **Phase:** Design / Build
- **Prompt:** "it would be really cool if this was all in one graph so you can see how the capital changes with buy / sells. what do you think?", then "let's have two charts then, good point", "on the capital chart, can we add like little points of interest [...] do these charts add a lot of complexity?", and "this needs to be mobile friendly too btw"
- **Why it works:** Asking "what do you think?" got a recommendation with a reason (capital is dollars in the thousands, stock is cases in the tens, so a dual axis misleads; stack two charts on a shared time axis instead) and a stated cost, which the user accepted in one line. The model then loaded the dataviz skill before writing any chart code and followed its checklist: chose a fixed color order, ran the palette validator for light and dark surfaces (neighbor color-blind separation 9.2), remapped colors so lemon is yellow and ice is blue and re-validated, and added the parts the skill treats as required (legend, hover crosshair, keyboard, table view).
- **Result:** Two charts with marker dots for trades and facility purchases, sparklines, and a game report, on desktop and phone. Screenshots at 1280 and 390 px, checked for horizontal overflow, showed the phone layout worked and caught a "0.5 cases" y-axis label that no unit test had asked about. It was fixed test-first (ticks must be whole numbers).
- **Lesson:** Ask for an opinion and the cost before approving a design. Load the rules skill first and make it run the validator, not eyeball it. Look at the rendered output at the smallest width.
