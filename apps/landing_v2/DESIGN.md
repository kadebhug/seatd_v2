---
name: Seatd Landing
description: The laminated host-stand chart — desk cream, timber, and live floor visibility for busy restaurants.
colors:
  canvas: "#e3ddd6"
  ink: "#141411"
  paper: "#f4efe6"
  signal: "#e54b2d"
  available: "#2f7d65"
  attention: "#d99a2b"
  timber: "#1c1410"
  danger: "#8f2f2f"
  muted: "color-mix(in srgb, #141411 58%, transparent)"
  line: "color-mix(in srgb, #141411 16%, transparent)"
typography:
  display:
    fontFamily: "var(--font-source-serif), \"Times New Roman\", serif"
    fontSize: "clamp(2.25rem, 5vw, 3.4rem)"
    fontWeight: 700
    lineHeight: 1.08
    letterSpacing: "-0.03em"
  headline:
    fontFamily: "var(--font-source-serif), \"Times New Roman\", serif"
    fontSize: "clamp(1.875rem, 4vw, 3rem)"
    fontWeight: 700
    lineHeight: 1.08
    letterSpacing: "-0.03em"
  title:
    fontFamily: "var(--font-archivo), Arial, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: "normal"
  body:
    fontFamily: "var(--font-archivo), Arial, sans-serif"
    fontSize: "17px"
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: "normal"
  label:
    fontFamily: "var(--font-archivo), Arial, sans-serif"
    fontSize: "0.9rem"
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: "normal"
  mono:
    fontFamily: "ui-monospace, SFMono-Regular, Consolas, monospace"
    fontSize: "0.78rem"
    fontWeight: 500
    lineHeight: 1.1
    letterSpacing: "-0.02em"
rounded:
  control: "10px"
  panel: "6px"
  feature: "4px"
spacing:
  sm: "8px"
  md: "16px"
  lg: "24px"
  section-y: "6rem"
  section-y-md: "8rem"
components:
  button-primary:
    backgroundColor: "{colors.signal}"
    textColor: "#ffffff"
    rounded: "{rounded.control}"
    padding: "0 18px"
    height: "48px"
  button-primary-hover:
    backgroundColor: "{colors.signal}"
    textColor: "#ffffff"
    rounded: "{rounded.control}"
    padding: "0 18px"
    height: "48px"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "0 18px"
    height: "48px"
  button-secondary-hover:
    backgroundColor: "color-mix(in srgb, #141411 6%, transparent)"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "0 18px"
    height: "48px"
  form-input:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    rounded: "{rounded.control}"
    padding: "10px 12px"
    height: "48px"
---

# Design System: Seatd Landing

## Overview

**Creative North Star: "The Host Chart"**

Seatd's landing page is not a SaaS dashboard floating in a device frame. It is the laminated chart clipped to a timber host stand — cream desk paper, walnut grain, a metal clip, and dry-erase table states that a busy floor team would actually read at a glance. Marketing copy sits on the same cream canvas as the product metaphor; the hero splits offer (left) from live proof (right) so the first viewport reads like a working host desk, not a template hero.

Density is editorial but operational: serif display headlines with tight tracking, Archivo body copy at 17px, and UI controls stamped at 10px radius like chart annotations. Color does the operational work — jade for available, coral for occupied and primary action, amber for attention — while ink and paper carry everything else.

**Key Characteristics:**

- Desk-cream canvas (#E3DDD6) with ink (#141411) type and paper (#F4EFE6) chart surfaces
- Source Serif 4 display + Archivo UI; mono for table numbers only
- Material depth from raster timber, laminate paper, and metal clipboard clip — not device chrome
- Floor-state semantics: jade / coral / amber fills on table tiles and legend dots
- Chart-stamp controls (10px radius), hairline dividers, double-ring focus
- Split hero: copy left, timber field + clipboard chart right

## Colors

The palette reads as a host stand under service lights — warm neutrals, ink-dark type, and three dry-erase marker colors for floor state.

### Primary

- **Signal Coral** (#E54B2D): Primary CTAs (`Book a Demo`), headline period accents, occupied table borders and fills, sales-walkthrough moment labels, featured pricing tint, and selection highlights. The accent earns its place on actions and live-state emphasis, not decoration.

### Secondary

- **Available Jade** (#2F7D65): Available table borders and fills, trust-list checkmarks, pricing include icons, and legend dots. Operational "open" state — never used for marketing emphasis.

### Tertiary

- **Attention Amber** (#D99A2B): Tables needing staff attention — borders, fills, legend, and a gentle pulse animation on attention tiles (respects `prefers-reduced-motion`).

### Neutral

- **Desk Cream** (#E3DDD6): Page canvas, header background, sticky demo bar, and focus-ring inner halo.
- **Chart Paper** (#F4EFE6): Laminate sheet surfaces, form fields, phone screen interior, pricing cards, and elevated panels on cream.
- **Service Ink** (#141411): Primary text, secondary button borders, clipboard chart type, and scrollbar thumb base.
- **Walnut Timber** (#1C1410): Hero and live-ops stage backgrounds (raster `seatd-host-timber.webp`), image caption gradients, and dark-stage table tile type.
- **Muted Ink** (58% ink mix): Supporting body copy via `text-muted` and placeholder tone.
- **Hairline** (16% ink mix): Borders, grid gutters, FAQ rules, and table dividers.
- **Form Danger** (#8F2F2F): Inline validation errors on demo form fields.

### Named Rules

**The Dry-Erase Rule.** Table and legend colors map 1:1 to floor semantics: jade = available, coral = occupied, amber = attention. Do not repurpose these hues for unrelated UI chrome.

**The Coral Sparingly Rule.** Signal coral appears on primary actions, the terminal period on display headlines, and live-state emphasis. It is not a general highlight color for icons, backgrounds, or section labels.

## Typography

**Display Font:** Source Serif 4 (with Times New Roman, serif fallback)  
**Body Font:** Archivo (with Arial, sans-serif fallback)  
**Label/Mono Font:** Archivo for labels; ui-monospace stack for table numbers

**Character:** Editorial confidence on the stand — serif headlines feel like stamped chart titles; Archivo keeps the sheet legible at service speed. Display type is always bold (700) with tight negative tracking; UI titles drop to semibold sans at normal tracking.

### Hierarchy

- **Display** (700, clamp 2.25–3.4rem / 36–54px at headline scale, line-height 1.08, −0.03em): Hero and section H1/H2 via `.display`. Ends with a coral period (`.period`). `text-wrap: balance` on headings.
- **Headline** (700, 1.875–3rem, line-height 1.08): Large section titles at `text-3xl`–`text-5xl` breakpoints.
- **Title** (600, 1.125–1.25rem, line-height ~1.35): Problem cards, role names, step titles, pricing tier names — sans semibold, normal tracking.
- **Body** (400, 17px, line-height 1.55): Default `body` size. Supporting copy often at `text-lg` (18px). Max comfortable measure ~42–58ch in marketing blocks.
- **Label** (600, 0.9rem): Form field labels, FAQ summaries, compare-table headers.
- **Mono** (500, 0.78rem, −0.02em): Table numbers inside `.table-tile strong` only.

### Named Rules

**The Coral Period Rule.** Display headlines that end a thought carry a `<span class="period">.</span>` in signal coral. The period is part of the headline voice, not optional punctuation styling.

**The Sans Inside the Chart Rule.** Floor UI chrome (legends, fixture labels, table sublabels) uses Archivo at small caps sizes (0.62–0.68rem, semibold, often uppercase). Serif stays on page-level display, call-sheet labels, and handwritten chart notes.

## Layout

The page uses a centered **shell** (max-width 1400px, horizontal padding `max(1.25rem, safe-area)`). Section anchors carry `scroll-margin-top: 5.5rem` for the sticky header.

**Hero split:** Full viewport height (`min-h-[100dvh]`) two-column grid on large screens — copy column ~0.92fr, timber/clipboard column ~1.18fr. Copy column uses asymmetric left padding to align with shell on wide viewports.

**Section rhythm:** Major sections use `py-24 md:py-32` (96px / 128px). Interior grids commonly `gap-10`–`gap-12`. Trust strip is compact (`py-6`) with a top/bottom hairline.

**Responsive grids:** Role cards (4-col), app tiles (2→4 col), pricing (3-col), venue mosaic (asymmetric 6-col spans), sales walkthrough (0.85 / 1.15 split). Mobile collapses table tile sublabels and tightens floor density.

**Sticky chrome:** Header (`h-16`, cream + bottom hairline) and optional sticky demo bar (`top: 3.5rem`, slides in on scroll).

## Elevation & Depth

Depth is **material-first**, not card-float. The cream page is flat; dimension comes from photographed timber, laminate paper texture, clipboard cast shadow, and tonal layering on chart surfaces. SaaS-style floating white cards on gray backgrounds are absent.

### Shadow Vocabulary

- **Clipboard lift** (`0 18px 50px` at 38% black + inset top highlight): Wooden clipboard backing behind chart sheets.
- **Clip hardware** (`0 8px 16px` at 35% black): Metal clip element, applied as a `drop-shadow()` filter on the clip SVG.
- **Sheet cast** (`0 2px 4px` at 18% + `0 10px 22px` at 22%): Laminate chart sheet lifting off the clipboard backing.
- **Focus ring** (`0 0 0 2px canvas, 0 0 0 4px ink`): Double-ring focus on interactive elements — no glow halos.

### Named Rules

**The Material Truth Rule.** Hero and product demos ground the chart in raster timber (`seatd-host-timber.webp`) and laminate paper (`seatd-chart-paper.webp`). Do not replace these with flat color blocks or CSS gradients when showing the signature clipboard composition. The clipboard backing samples a different crop of the timber plank than the field behind it and is lifted warmer by a `screen` pass, so it reads as a separate piece of wood rather than a window onto the desk.

**Clip exception.** The metal clip is the one vector element in the composition — a hand-built inline SVG (`components/clipboard-clip.tsx`). `seatd-clipboard-clip.webp` is unusable as a layer: it is an opaque 200×80 crop with no alpha, carrying label text and mock background. Polished metal is also the material vector renders most convincingly, provided the chrome ramp stays **non-monotonic** (light → dark → light → dark). A plain light-to-dark gradient is what made the previous CSS clip read as plastic. If a clean cut-out clip raster with alpha is ever produced, swap it back.

**The Flat Canvas Rule.** Marketing sections on cream use hairline borders and paper fills for separation. Reserve drop shadows for the clipboard object and phone shell — not general content cards.

## Shapes

Controls and stamps use **10px corner radius** (`--radius-control`) — chart-annotation rounds, not pill buttons. Panels and clipboard sheets use tighter geometry: 6px panel radius (`--radius-panel`, including the laminate chart sheet), 4px feature radius on square table tiles. Circular tables are true `border-radius: 50%`.

Borders are 1px hairlines (`--seatd-line`) or 2px on table tiles. The clipboard silhouette uses asymmetric radius (`3px 3px 8px 8px`). Section grids sometimes expose gutters as 1px `bg-line` gaps between cream or paper cells — no outer card wrapper.

## Components

### Buttons

- **Shape:** Chart-stamp rounded rectangle (10px radius), 48px min-height
- **Primary:** Signal coral fill, white semibold label (0.95rem), 18px horizontal padding. Hover: `brightness(0.96)`. Active: `scale(0.98)`. Class: `.cta-primary`
- **Secondary:** Transparent fill, 1px ink border, inherits text color. Hover: 6% ink wash. Class: `.cta-secondary`
- **Focus:** Double-ring via `.focus-ring` / `:focus-visible`

### Form Fields

- **Style:** Paper surface, 1px hairline border, 10px radius, 48px min-height, 10×12px padding
- **Label:** 0.9rem semibold, 8px gap above control
- **Focus:** Double-ring focus shadow
- **Error:** Danger color at 0.85rem below field

### Navigation

- **Header:** Sticky cream bar, lockup left, centered text links at 0.95rem (`text-fg/80` → full fg on hover), primary CTA right
- **Mobile:** Full-height overlay menu from below header; 2xl semibold link stack
- **In-page:** Sales call-sheet beats link to section anchors with scroll-spy active state (6% coral wash)

### Cards / Containers

- **Chart sheet:** Paper color + laminate raster, ink type, 1rem padding; floor canvas borderless inside sheet
- **Call sheet:** Paper + laminate, hairline outer border, beat rows separated by 12% ink mix top borders
- **Pricing tier:** Paper surface, hairline border; featured tier gets 5% coral fill and 32% coral border mix
- **Ticket / surface panels:** Paper on cream with hairline border, 22px padding

### Table Tiles (signature)

- **Shape:** 2px border; circle (50%, min-height 46px) or 4px rounded rect (min-height 44px, 36px mobile)
- **States:** Available (18% jade mix), occupied (18% coral mix), attention (22% amber mix + pulse). Active table gets a double ring, never a transform — `attention-pulse` animates `transform` and would override it
- **Type:** Mono table number, 0.62rem semibold status sublabel (hidden on mobile). All labels clamp with `text-overflow: ellipsis` — nothing bleeds past a tile
- **Dark stage:** On timber (`.desk-stage`), fills deepen to ~28–30% mix; type flips to paper color

### Clipboard Hero (signature)

- **Composition:** Timber field → clipboard board (timber raster + shadow) → metal clip (inline SVG, centered, its grip lip overlapping the sheet top) → chart sheet with live floor, legend, and italic chart note
- **Legend:** 0.55rem dots matching state colors; uppercase micro labels at 0.62rem
- **Tilt:** The whole clipboard sits at `rotate(-0.6deg)`. Photographed objects are never axis-aligned; the tilt carries more of the "real object" read than gradient tuning does
- **Chart note:** Anchored to the floor canvas's percentage space, not the sheet, and parked in clear canvas per service step. It sits above every tile by design, so placement is what keeps it off them

### Phone Shell

- Near-black `#1A1A16` rounded device frame (36px radius) wrapping paper screen (26px inner radius) for guest-assist demo — the one allowed "device" frame, treated as prop not layout chrome.

## Do's and Don'ts

### Do:

- **Do** keep the page background desk cream and let paper surfaces carry elevated content.
- **Do** end major display headlines with a coral period on the closing beat.
- **Do** use jade, coral, and amber exclusively for available, occupied, and attention table semantics.
- **Do** stamp controls at 10px radius with 48px touch targets.
- **Do** pair Source Serif display with Archivo for all marketing and form UI.
- **Do** ground product proof in the clipboard + timber + laminate paper material stack.

### Don't:

- **Don't** float marketing sections as drop-shadow cards on a gray SaaS canvas.
- **Don't** use pill-shaped or fully rounded buttons for primary actions.
- **Don't** introduce additional accent colors beyond signal, available, and attention for floor-state meaning.
- **Don't** use display serif for dense UI labels inside the floor canvas — keep those in Archivo.
- **Don't** swap raster timber or paper for flat CSS approximations on the signature hero clipboard.
- **Don't** put status words inside circular table tiles — a round 2-top has no room, and colour plus the legend already carry the state.
