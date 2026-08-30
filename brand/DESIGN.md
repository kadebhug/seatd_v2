# Seatd Design System

Version 1.0  
Brand promise: **Every table, in sync.**

This document defines the shared visual language for Seatd's marketing site, platform admin, restaurant owner app, waiter app, entrance display, and guest QR assist experience. It is the implementation reference for design and engineering.

## 1. Brand idea

Seatd is calm front-of-house infrastructure. It makes table availability and guest assistance visible without making the technology the centre of the dining experience.

The identity should feel:

- hospitable, not clinical;
- precise, not technical for its own sake;
- calm under pressure;
- modern without looking like a generic SaaS product;
- premium enough for a considered restaurant, practical enough for daily service.

The symbol combines two simplified seat/table modules. Their opposing curves create an implied `S` and a continuous path between guest and staff. The coral upper form is the live signal; the lower form is the stable operating surface.

## 2. Logo system

### Master variants

| Context | Asset | Use |
| --- | --- | --- |
| Light or warm background | `core/seatd-lockup-color.svg` | Default horizontal logo |
| Bone field | `core/seatd-lockup-on-bone.svg` | Controlled marketing and splash use |
| Ink or dark photography | `core/seatd-lockup-reverse.svg` | Dark surfaces |
| Small square placement | `core/seatd-symbol-color.svg` | App bars, avatars, compact navigation |
| Small square on Ink | `core/seatd-symbol-reverse.svg` | Dark compact surfaces |
| One-colour production | `core/seatd-symbol-mono-ink.svg` or `core/seatd-symbol-mono-bone.svg` | Emboss, etch, vinyl, receipt printing |

The symbol and wordmark are one brand. Owner, waiter, display, guest, and platform products must not receive different symbols or colours. Differentiate the surface with a written role label in the interface.

### Clear space

Use the diameter of the upper circular element as `x`.

- Symbol: keep at least `0.75x` clear space on all sides.
- Horizontal lockup: keep at least `1x` above and below, and `1.5x` at both ends.
- Never place table-state dots, QR finder corners, or notification badges inside the clear space.

### Minimum sizes

| Asset | Digital minimum | Print minimum |
| --- | ---: | ---: |
| Symbol | 20 px | 7 mm |
| Horizontal lockup | 112 px wide | 32 mm wide |
| QR table marker symbol | 32 px | 10 mm |

At 16 px, use the supplied favicon artwork. Do not resize the full lockup into favicon dimensions.

### Logo rules

Do:

- use supplied files without redrawing them;
- use the colour mark on Bone, white, or very light photography;
- use the reverse mark on Ink or sufficiently dark photography;
- use monochrome artwork when production permits only one ink;
- preserve aspect ratio and clear space.

Do not:

- recolour the two halves with restaurant-specific colours;
- use Jade or Amber inside the master mark;
- add a role name into the mark;
- rotate, skew, outline, bevel, shadow, or animate the geometry independently;
- put the colour mark on red, orange, or visually busy photography;
- treat the symbol as a table-status indicator.

## 3. Colour

### Core palette

| Token | Hex | Purpose |
| --- | --- | --- |
| Ink | `#141411` | Primary text, dark fields, stable brand surface |
| Bone | `#F3EEE4` | Primary warm background |
| Signal | `#F05A3C` | Brand accent, primary action, occupied state when explicitly labelled |
| Jade | `#2F7D65` | Available state and success |
| Amber | `#D99A2B` | Attention state and caution |
| White | `#FFFFFF` | High-contrast surface and reverse text where required |

The operational state mapping is fixed:

| State | Colour | Required companion |
| --- | --- | --- |
| Available | Jade | `Available` label and/or available icon |
| Occupied | Signal | `Occupied` label and/or occupied icon |
| Attention | Amber | `Attention` label and/or attention icon |

State must never be communicated by colour alone. Pair colour with a label, shape, icon, or pattern. Signal is both the brand accent and the occupied colour, so interface context must always make the meaning explicit.

### Colour balance

- Marketing: approximately 60% Bone, 30% Ink, 10% Signal.
- Operational apps: neutral surfaces first; state colours appear only where state is meaningful.
- Entrance display: maximise contrast and legibility at distance. Avoid decorative Signal accents near table markers.
- Guest assist: use Signal for the primary request action and Amber for a pending request. Use Jade only for a confirmed or resolved outcome.

No gradients are part of the core identity.

## 4. Typography

### Families

- **Instrument Sans** — primary interface, marketing, display, and word-led compositions.
- **DM Mono** — table identifiers, pairing codes, timestamps, technical metadata, small operational labels.
- System fallback: `Inter`, `Arial`, `sans-serif` for Instrument Sans and `ui-monospace`, `SFMono-Regular`, `Consolas`, `monospace` for DM Mono.

Do not use DM Mono for paragraphs or primary controls.

### Type scale

| Token | Size / line height | Weight | Use |
| --- | --- | --- | --- |
| Display XL | `64 / 64` | 700 | Marketing hero only |
| Display | `48 / 52` | 700 | Section headlines, large kiosk summary |
| H1 | `36 / 42` | 700 | Page title |
| H2 | `28 / 34` | 650 | Section title |
| H3 | `22 / 28` | 650 | Card or panel title |
| Body L | `18 / 28` | 450 | Guest and marketing body |
| Body | `16 / 24` | 450 | Default interface copy |
| Label | `14 / 18` | 650 | Controls and status labels |
| Utility | `12 / 16` | 600 | Codes, timestamps, compact metadata |

Use sentence case. Reserve uppercase with increased tracking for short role labels, kiosk wayfinding, and utility metadata.

## 5. Layout and shape

- Base spacing unit: `4 px`.
- Common spacing: `8, 12, 16, 24, 32, 48, 64, 96`.
- Minimum touch target: `44 × 44 px`; use `48 × 48 px` for waiter and owner controls.
- Corner radii: `8 px` controls, `16 px` panels, `24 px` feature surfaces.
- Use thin `1 px` separators with Ink at 12–18% opacity.
- Prefer flat surfaces. Use shadows only for modal elevation or draggable tables.
- Floor-plan geometry should be visually dominant; surrounding chrome should recede.

Seatd should not become a grid of cards. Use panels only when they establish a real grouping or interaction boundary.

## 6. Interface behaviour

### Motion

- Fast state acknowledgement: `120–180 ms`.
- Panel transitions: `220–320 ms`.
- Use ease-out for entries and ease-in for exits.
- Never animate a table state in a way that delays recognition.
- Attention may use a restrained pulse, but it must stop when reduced motion is enabled.
- Marketing motion may be cinematic; operational motion must be functional.

### Status language

Use the exact product terms: **Available**, **Occupied**, and **Attention**. Do not substitute “free”, “busy”, or “alert” inside staff and display products unless product requirements explicitly change.

### Accessibility

- Target WCAG 2.2 AA for interactive surfaces.
- Do not place Bone body text on Amber or Signal.
- Table states require text/icon redundancy.
- Provide visible keyboard focus on web and desktop.
- Respect reduced motion and text scaling.
- Guest QR actions must remain usable at 200% zoom and on narrow mobile screens.
- Entrance display labels must remain readable from the venue's expected viewing distance.

## 7. Product surfaces

### Marketing site

Use the horizontal lockup in the header and footer. Let Bone and Ink dominate; reserve Signal for the principal call to action and carefully selected emphasis. Restaurant imagery should be architectural, atmospheric, and people-light rather than generic hospitality stock photography.

### Platform admin

Use the colour symbol or lockup on a neutral surface. Keep the admin product operational and restrained. Destructive or disabled states must use semantic error styling rather than borrowing the brand mark.

### Owner app

Use the shared app icon. Owner identity appears as a text label in the title bar or sign-in context. Layout editing handles may use Signal; table state colour remains semantic.

### Waiter app

Prioritise speed, touch size, contrast, and state recognition. The logo should be limited to login, lock, and empty/loading moments. Do not consume active service space with a large brand header.

### Entrance display

Use the reverse lockup on Ink during pairing, loading, or idle moments. During live floor display, keep branding small and fixed so it cannot be mistaken for a table or state marker.

### Guest QR assist

Use the colour symbol above the restaurant and table context. Seatd remains the service layer; the restaurant name should be more prominent than Seatd after successful table lookup. Primary request actions use Signal. Pending uses Amber. Resolved uses Jade.

## 8. Platform asset map

| Target | Supplied asset | Destination guidance |
| --- | --- | --- |
| Browser favicon | `web/favicon.ico`, `web/favicon.svg` | Web public root and document head |
| PWA | `web/pwa-192.png`, `web/pwa-512.png` | Web manifest icons |
| PWA maskable | `web/maskable-192.png`, `web/maskable-512.png` | Manifest entries with `purpose: maskable` |
| Apple web icon | `web/apple-touch-icon.png` | `<link rel="apple-touch-icon">` |
| Social preview | `web/og-brand.png` | Open Graph and social metadata |
| Flutter shared source | `flutter/app-icon-1024.png` | Input to platform icon tooling |
| Android adaptive foreground | `flutter/android/ic_launcher_foreground.png` | Adaptive icon foreground |
| Android background | `flutter/android/ic_launcher_background.svg` | Convert/use as Bone adaptive background |
| Android themed icon | `flutter/android/ic_launcher_monochrome.svg` | Android 13+ monochrome layer |
| iOS | `flutter/ios/AppIcon-1024.png` | App Store icon source; opaque |
| Windows | `flutter/windows/seatd.ico` | Windows runner icon |
| macOS | `flutter/macos/seatd-*.png` | Populate the macOS AppIcon set |
| Linux | `flutter/linux/seatd.svg`, `seatd-512.png` | Desktop launcher and packaging |
| Role splash screens | `surfaces/*-splash.svg` and `.png` | Owner, waiter, display, guest assist, platform admin |

## 9. Implementation tokens

Use `tokens/seatd.tokens.json` as the machine-readable source for the core palette and radii.

CSS baseline:

```css
:root {
  --seatd-ink: #141411;
  --seatd-bone: #f3eee4;
  --seatd-signal: #f05a3c;
  --seatd-available: #2f7d65;
  --seatd-attention: #d99a2b;
  --seatd-white: #ffffff;
}
```

Flutter baseline:

```dart
abstract final class SeatdColors {
  static const ink = Color(0xFF141411);
  static const bone = Color(0xFFF3EEE4);
  static const signal = Color(0xFFF05A3C);
  static const available = Color(0xFF2F7D65);
  static const attention = Color(0xFFD99A2B);
  static const white = Color(0xFFFFFFFF);
}
```

## 10. Asset governance

- `core/` files are master assets. Changes require a brand version increment.
- Generated raster sizes may be regenerated from the SVG masters.
- Do not optimise or trace the SVG files through an online service.
- Keep logo files in version control alongside the consuming application.
- Record changes in the product repository and update this document when colour, geometry, typography, or naming changes.
- Use customer-facing name **Seatd** even where legacy code still uses TableReady.
