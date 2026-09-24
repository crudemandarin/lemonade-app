# Event backdrops: design

Looping, full-viewport background animations paired with the random events (SPEC rules 20-21, DESIGN §6). Status: **design only**. A throwaway prototype exists in the working tree (see §11) and should be reworked to match this doc.

## 1. Goals and non-goals

- One ambient scene per active event, looping forever until the event expires.
- Covers the **entire viewport**, at any size, behind all content.
- **Low detail, low contrast.** It sets a mood; it never competes with prices, buttons, or the day report.
- Cheap: compositor-only CSS, no canvas, no JS per frame, no new dependencies.
- Decorative only: no information lives in it. The events banner stays the source of truth.
- Non-goals: sound, interactivity, per-day variation, animating on day-report or game-over screens.

## 2. Events covered

Keys match `lemonade-api/internal/domain/config.go`. Adding an event to the table needs a scene here, or it gets none (silently, no error).

| Key | Name | Effect | Days | Mood | Direction |
|---|---|---|---|---|---|
| `heat_wave` | Heat Wave | lemonade ×1.4, ice ×1.3 | 2 | hot, bright | good for selling |
| `rainy_week` | Rainy Week | lemonade ×0.75 | 3 | grey, damp | bad for selling |
| `lemon_blight` | Lemon Blight | lemon ×1.7 | 3 | sickly, decaying | bad for buying |
| `sugar_glut` | Sugar Glut | sugar ×0.7 | 2 | abundant, heavy | good for buying |
| `holiday` | Holiday | lemonade ×1.35 | 1 | festive | good for selling |
| `cup_shortage` | Cup Shortage | cup ×1.5 | 2 | scarce, empty | bad for buying |

## 3. Layering rules

- **Placement:** one `app-event-backdrop` in the app shell (`app.component.html`), above the nav bar in the template, `position: fixed; inset: 0; z-index: -1`. It shows on every page (sign-in and home have no game, so no events, so nothing renders).
- **Input:** the active `GameEvent[]` from `GameStore.game()`. Nothing else; the component is presentational.
- **Cap of 2 scenes.** Events can overlap (only same-key copies are blocked, DECISIONS §4). Take known keys, sort by `daysLeft` descending (longest-running first, ties keep server order), draw the first two. Others are ignored.
- **Same-slot conflicts.** Two scenes can both want the top of the screen or the falling layer, so each scene declares which slots it uses (§5); if the top two collide, still draw both (they are faint and translucent) but the second gets `opacity: 0.6` of its own so the first reads.
- **Whole layer opacity 0.55**, individual particles 0.35-0.7. Both tints and particles are semi-transparent so cards (opaque `--surface`) always sit cleanly on top.
- **Fade-in** 1.2 s when a scene appears; fade-out is not needed (the scene is removed on expiry, at a day boundary the player has just clicked through).
- **Bankrupt / game over:** backdrop hidden (`status === 'bankrupt'`).

## 4. Shared mechanics

- **Coverage:** every position and size is in `%`, `vw`, `vh`, `vmin`, `vmax`, so the scene scales to any viewport with no JS resize handling. Falling particles travel from `-12vh` to `112vh`.
- **Particles:** N absolutely positioned elements, each driven by four CSS variables from a pure function `vars(i, n, minDur, spread)`:
  - `--x`: `(i + 0.5) / n × 100%` (even spread, no clumps, no randomness)
  - `--dur`: `minDur + frac(i × 0.382) × spread`
  - `--delay`: `-frac(i × 0.618) × dur` (negative, so the scene is already mid-loop when it appears)
  - `--j`: `frac(i × 0.618)` (0-1 jitter for size or vertical position)
- **Deterministic:** golden-ratio arithmetic, no `Math.random()`, so it is testable and identical between renders.
- **Seamless loop:** every keyframe starts and ends off-screen or at the same visual state; `animation-iteration-count: infinite`, `linear` for falling, `ease-in-out` for pulses and sways.
- **Performance budget:** at most **~40 animated elements per scene**, only `translate`, `rotate`, `scale`, `opacity` animated (compositor properties), `will-change` only on falling particles. No `filter`, no `box-shadow`, no blur. Rain and confetti are the heaviest scenes at 36 and 32 particles.
- **Reduced motion** (`prefers-reduced-motion: reduce`): all animation off. Keep the static tint and, for heat wave, the static sun, as a still cue. Particles hidden.
- **Dark mode:** same colours; the tints are low-alpha so they work on both `--bg` values. Sugar cubes and confetti are the only light-on-dark risk and are kept at ≤0.6 opacity.
- **Accessibility:** `aria-hidden="true"`, `pointer-events: none`, no focusable content. Text contrast is unaffected because cards are opaque; the nav bar and stats strip are opaque too.
- **Unknown or new keys:** no scene, no error. Cheap to add later.

## 5. Scenes

Each scene = a **tint** (full-viewport gradient wash, static) + **motion** (looped elements). Colours are RGB with alpha; final values are tuned by eye in both themes.

### 5.1 Heat wave: `heat_wave`

- **Tint:** vertical wash, `rgb(250 170 30 / 0.10)` at the top to `rgb(250 120 30 / 0.16)` at the bottom.
- **Sun:** a ~52vmax circle centred off the **top-right corner** (`top/right: -18vmax`), radial gradient yellow core to transparent. Pulses `scale 1 → 1.07 → 1` over 7 s. A faint ray ring (`repeating-conic-gradient`, 24° pitch, masked to an annulus) rotates once per 90 s.
- **Haze bands:** 3 wide soft horizontal bands (12vh tall, warm translucent gradient) that rise from below the viewport to above it over 9-13 s, drifting 4vw sideways: reads as heat shimmer without any SVG filter.
- **Slots:** top-right corner, full-height haze. Does not use the falling layer.
- **Reduced motion:** static tint + static sun.

### 5.2 Rainy week: `rainy_week`

- **Tint:** `rgb(71 85 105 / 0.22)` at the top to `rgb(100 116 139 / 0.10)` at the bottom (grey-blue, darker overhead).
- **Clouds:** 2 wide soft radial ellipses (55vw and 45vw wide, ~22vh tall) drifting left to right across the top over 80 s and 110 s, the second offset by half a cycle.
- **Rain:** 36 thin streaks (2 px × 7vh, gradient from transparent to light blue), tilted 10°, falling `-10vh → 112vh` with a 12vh leftward drift, 1.1-2.0 s each: fast, so it feels like rain rather than confetti.
- **Slots:** top edge (clouds) and falling layer.
- **Optional later:** a single lightning flash (full-viewport white at 0.15 opacity for 150 ms every ~20 s). Not in v1.

### 5.3 Lemon blight: `lemon_blight`

- **Tint:** olive **vignette**, `radial-gradient(ellipse at center, transparent 40%, rgb(120 113 30 / 0.25))`, so the centre (content) stays clean and the edges look sickly.
- **Leaves:** 12 leaf shapes (`border-radius: 0 100% 0 100%`, ~3vmin, min 18 px), colours brown `#a16207`, dark brown `#854d0e`, sickly green `#65a30d` in a 3-cycle. They fall `-12vh → 112vh` over 16-26 s, swaying ±6-7vw at mid-fall and spinning ~500° (direction varies by colour).
- **Slots:** falling layer.

### 5.4 Sugar glut: `sugar_glut`

- **Tint:** `rgb(147 197 253 / 0.08)` to `rgb(255 255 255 / 0.10)` (cool, milky).
- **Cubes:** 12 copies of `assets/resources/sugar.svg`, size `4vmin + j × 4vmin` (min 28 px), 0.6 opacity, falling slowly (18-30 s), gentle 3vw sway, 300° spin. Slow and many = "too much supply".
- **Slots:** falling layer.
- **Optional later:** a soft white pile along the bottom edge that grows over the loop.

### 5.5 Holiday: `holiday`

- **Tint:** `rgb(244 114 182 / 0.12)` to `rgb(167 139 250 / 0.10)` (pink to lilac).
- **Bunting:** a horizontal line (2 px, muted) across the full top edge with 12 triangular flags (`clip-path` triangles, ~4.5vw, min 22 px) evenly spaced, cycling red/yellow/blue/green, each swinging ±8° about its top with a 4-6 s ease-in-out loop and staggered delays.
- **Confetti:** 32 small rectangles (~1.1 × 2 vmin, min 7 × 12 px), same four colours, falling 7-12 s with ±3-4vw sway and 720° spin.
- **Slots:** top edge (bunting) and falling layer.
- **Optional later:** themed variants by calendar date (snow, pumpkins), corner fireworks on the single day it lasts.

### 5.6 Cup shortage: `cup_shortage`

- **Tint:** faint red **vignette**, `radial-gradient(ellipse at center, transparent 45%, rgb(220 38 38 / 0.10))`.
- **Cups:** 10 copies of `assets/resources/cup.svg`, size `6vmin + j × 4vmin` (min 36 px), scattered at fixed positions (x from `--x`, y at `8% + j × 78%`). Each **vanishes**: fades in to 0.7 over the first 20% of its 9-15 s loop, holds, then fades out while tilting -18° and rising 3vh. The idea is shelves that keep emptying.
- **Slots:** whole viewport, not tied to the falling layer.

## 6. Component design

`shared/event-backdrop/`: standalone, presentational, files `event-backdrop.component.{ts,html,scss}` (+ spec).

- `input events: GameEvent[]` (default `[]`).
- `scenes = computed(...)`: known keys, longest `daysLeft` first, `slice(0, 2)`.
- Template: `@for` over scenes with `@switch` on the key; each scene renders its own particles with `@for` over a precomputed index array, `[style]="vars(i, n, minDur, spread)"` (object binding sets the custom properties).
- Only `.scene`, tint, and particle elements; no host layout, `:host { display: contents }`.
- `app.component` passes `store.game()?.events ?? []`, and passes `[]` when bankrupt.
- Adding a scene = one `@case`, one SCSS block, one entry in the known-keys list.

## 7. Styles

- All motion is `@keyframes` on `translate`, `rotate`, `scale`, `opacity` (individual transform properties; supported by current evergreen browsers, which the PWA already assumes).
- The SCSS for six scenes is ~4 kB, above Angular's default 2 kB per-component style budget (the prototype tripped it). Decision: **raise the `anyComponentStyle` warning budget in `angular.json` to 6 kB**, rather than splitting styles, since the file is one cohesive unit.
- Constants (counts, opacity, cap) live at the top of the `.ts` file, so tuning is one place.

## 8. Testing

- **Unit (component):**
  - no events renders no scene;
  - each known key renders its scene container;
  - unknown key renders nothing;
  - more than two events renders exactly two, longest `daysLeft` first;
  - `vars()` is deterministic and its outputs stay in range (`--x` within 0-100%, negative delay, `--j` within 0-1).
- **Shell:** `AppComponent` renders the backdrop and passes the game's events; bankrupt passes none.
- **Manual:** each event visually in light and dark mode, at phone width and a large desktop, with `prefers-reduced-motion` on and off, and DevTools Performance to confirm no layout or paint work per frame. To trigger an event without waiting for a roll, temporarily give the fixtures/dev game a chosen event.

## 9. Trade-offs

- **CSS-only vs canvas:** canvas would allow richer effects, but costs JS work every frame and battery on mobile; CSS keyframes on compositor properties are close to free and are enough for "low detail".
- **Deterministic vs random particles:** uniform golden-ratio spacing avoids clumping and makes tests trivial, at the cost of a slightly regular pattern; at these counts and opacities it is not noticeable.
- **Cap of 2 scenes:** more would muddy the screen and multiply the cost; overlap is rare (25% chance/day, six events, 1-3 days).
- **Fixed layer vs per-page:** one backdrop in the shell means no per-page wiring and no unmount/remount when navigating.
- **Prefers-reduced-motion keeps the tint:** the mood still communicates the event, so reduced-motion users lose motion only.

## 10. Open questions

1. Cap at 2 scenes, or 1 (the strongest only)? Recommendation: 2.
2. Should the falling layers (rain, sugar, confetti, leaves) stay clear of the centre column to avoid crossing text? Recommendation: no, they're faint and behind opaque cards.
3. Include the optional extras (lightning, sugar pile, fireworks) in v1? Recommendation: no, follow up.

## 11. State of the prototype

Uncommitted, not yet reviewed against this design:

- `lemonade-web/src/app/shared/event-backdrop/` (component, all six scenes)
- `app.component.{ts,html}` wired to it
- The build passes; the only warning is the style budget (§7 fix not yet applied).

Not done yet: raising the budget, component and shell specs, bankrupt handling in the shell, visual review in light/dark and reduced-motion, and doc updates (SPEC, DESIGN §7, PLAN slice 6).
