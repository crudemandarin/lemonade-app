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
- **Result:** It found that the frontend contradicts DESIGN: all warehouses share one level, and only building counts are per resource, so the routes take `warehouse`/`production` rather than six kinds. It built to the frontend and recorded the deviation as DECISIONS #3 and flagged it in the summary, rather than silently guessing (CLAUDE.md's "ask, don't guess" rule).
- **Lesson:** When a client already exists, point the model at it by name. Expect the docs to be stale in places, and require deviations to be written down.

### 7. Test-first caught an AI-written float bug; the AI's own tests were wrong too
- **Phase:** Build
- **Prompt:** (same session as #6; the CLAUDE.md workflow "write failing tests that encode the acceptance criteria, then implement" did the work)
- **Why it works:** The rounding table test (`price 100 -> bid 90 / ask 110`) came straight from SPEC rule 7. It failed with an ask of 111 because `100 * 1.1 = 110.00000000000001` and `ceil` rounded up. A second failure was the model's test under-funding itself (6 buildings x $200 > $1,000), so the test, not the code, was fixed.
- **Result:** A `snap` helper that rounds away float noise before floor/ceil (DECISIONS #5), plus a test that asserts bid < ask and bid >= 1 for every price from $1 to $500.
- **Lesson:** Specify examples from the spec, not from the implementation. When a test fails, decide which side is wrong before touching either.

### 8. Reviewing its own test for blast radius, then verifying against the real stack
- **Phase:** Review
- **Prompt:** (no user prompt; the model's own checks after the Postgres store was written)
- **Why it works:** The model noticed that its integration test ran `DELETE FROM games/users` against whatever `DATABASE_URL` pointed at, which could wipe a developer's database, and scoped the cleanup to `pguser%` rows before ever running it. It then ran the test against the live Docker Postgres (credentials sourced from `.env` without printing them), including 20 concurrent `Mutate` calls to prove the `FOR UPDATE` lock prevents lost updates, and finally started the server on a spare port and played a full day with curl.
- **Result:** A test that is safe to run on a shared DB, a lock behavior proven rather than assumed, and an end-to-end run showing the real JSON shapes and error bodies.
- **Lesson:** Ask "what does this test destroy if pointed at the wrong place?" before running it. Prefer one real-stack run over more mocks, and use a spare port so it doesn't collide with the running containers.
