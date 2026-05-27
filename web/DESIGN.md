---
version: alpha
name: Coinbase-design-analysis
description: An institutional-grade crypto exchange whose marketing surfaces read like a quietly-confident financial-services brand. The base canvas is pure white; Coinbase Blue (`#0052ff`) is the single brand voltage, used scarcely on primary CTAs, signature glyphs, and inline accent moments. Type runs Coinbase's licensed CoinbaseDisplay (display) and CoinbaseSans (body) at modest weights — display sits at weight 400 not 700, signaling editorial calm rather than fintech-bombastic. Page rhythm rotates between bright white sections, soft gray elevation bands, and full-bleed dark editorial heroes (`#0a0b0d`) carrying product-ui mockup cards. Iconography is geometric and minimal; depth comes from card-on-card layering, never decorative shadows.

colors:
  primary: "#0052ff"
  primary-active: "#003ecc"
  primary-disabled: "#a8b8cc"
  ink: "#0a0b0d"
  body: "#5b616e"
  body-strong: "#0a0b0d"
  muted: "#7c828a"
  muted-soft: "#a8acb3"
  hairline: "#dee1e6"
  hairline-soft: "#eef0f3"
  canvas: "#ffffff"
  surface-soft: "#f7f7f7"
  surface-card: "#ffffff"
  surface-strong: "#eef0f3"
  surface-dark: "#0a0b0d"
  surface-dark-elevated: "#16181c"
  on-primary: "#ffffff"
  on-dark: "#ffffff"
  on-dark-soft: "#a8acb3"
  semantic-up: "#05b169"
  semantic-down: "#cf202f"
  accent-yellow: "#f4b000"

typography:
  display-mega:
    fontFamily: "'Coinbase Display', -apple-system, system-ui, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif"
    fontSize: 80px
    fontWeight: 400
    lineHeight: 1.0
    letterSpacing: -2px
  display-xl:
    fontFamily: "'Coinbase Display', sans-serif"
    fontSize: 64px
    fontWeight: 400
    lineHeight: 1.0
    letterSpacing: -1.6px
  display-lg:
    fontFamily: "'Coinbase Display', sans-serif"
    fontSize: 52px
    fontWeight: 400
    lineHeight: 1.0
    letterSpacing: -1.3px
  display-md:
    fontFamily: "'Coinbase Display', sans-serif"
    fontSize: 44px
    fontWeight: 400
    lineHeight: 1.09
    letterSpacing: -1px
  display-sm:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 36px
    fontWeight: 400
    lineHeight: 1.11
    letterSpacing: -0.5px
  title-lg:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 32px
    fontWeight: 400
    lineHeight: 1.13
    letterSpacing: -0.4px
  title-md:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 18px
    fontWeight: 600
    lineHeight: 1.33
    letterSpacing: 0
  title-sm:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 16px
    fontWeight: 600
    lineHeight: 1.25
    letterSpacing: 0
  body-md:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: 0
  body-strong:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 16px
    fontWeight: 700
    lineHeight: 1.5
    letterSpacing: 0
  body-sm:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: 0
  caption:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: 0
  caption-strong:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 12px
    fontWeight: 600
    lineHeight: 1.5
    letterSpacing: 0
  number-display:
    fontFamily: "'Coinbase Mono', 'Coinbase Sans', monospace"
    fontSize: 18px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: 0
  button:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 16px
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: 0
  nav-link:
    fontFamily: "'Coinbase Sans', sans-serif"
    fontSize: 14px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: 0

rounded:
  none: 0px
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  xl: 24px
  pill: 100px
  full: 9999px

spacing:
  xxs: 4px
  xs: 8px
  sm: 12px
  base: 16px
  md: 20px
  lg: 24px
  xl: 32px
  xxl: 48px
  section: 96px

components:
  top-nav-light:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.nav-link}"
    height: 64px
  top-nav-on-dark:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.nav-link}"
    height: 64px
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: 12px 20px
    height: 44px
  button-primary-active:
    backgroundColor: "{colors.primary-active}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.pill}"
  button-primary-disabled:
    backgroundColor: "{colors.primary-disabled}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.pill}"
  button-secondary-light:
    backgroundColor: "{colors.surface-strong}"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: 12px 20px
    height: 44px
  button-secondary-dark:
    backgroundColor: "{colors.surface-dark-elevated}"
    textColor: "{colors.on-dark}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: 12px 20px
    height: 44px
  button-outline-on-dark:
    backgroundColor: transparent
    textColor: "{colors.on-dark}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: 11px 19px
    height: 44px
  button-tertiary-text:
    backgroundColor: transparent
    textColor: "{colors.primary}"
    typography: "{typography.button}"
  button-pill-cta:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.button}"
    rounded: "{rounded.pill}"
    padding: 16px 32px
    height: 56px
  hero-band-dark:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.display-mega}"
    padding: 96px
  hero-band-light:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.display-mega}"
    padding: 96px
  product-ui-card-dark:
    backgroundColor: "{colors.surface-dark-elevated}"
    textColor: "{colors.on-dark}"
    rounded: "{rounded.xl}"
    padding: 32px
  product-ui-card-light:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    rounded: "{rounded.xl}"
    padding: 32px
  feature-card:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.title-md}"
    rounded: "{rounded.xl}"
    padding: 32px
  asset-row:
    backgroundColor: transparent
    textColor: "{colors.ink}"
    typography: "{typography.body-md}"
    padding: 16px 0
  price-up-cell:
    backgroundColor: transparent
    textColor: "{colors.semantic-up}"
    typography: "{typography.number-display}"
  price-down-cell:
    backgroundColor: transparent
    textColor: "{colors.semantic-down}"
    typography: "{typography.number-display}"
  pricing-tier-card:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 32px
  pricing-tier-featured:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.body-md}"
    rounded: "{rounded.xl}"
    padding: 32px
  cta-band-dark:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.on-dark}"
    typography: "{typography.display-lg}"
    padding: 96px
  text-input:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body-md}"
    rounded: "{rounded.md}"
    padding: 14px 16px
    height: 48px
  search-input-pill:
    backgroundColor: "{colors.surface-strong}"
    textColor: "{colors.ink}"
    typography: "{typography.body-md}"
    rounded: "{rounded.pill}"
    padding: 12px 20px
    height: 44px
  badge-pill:
    backgroundColor: "{colors.surface-strong}"
    textColor: "{colors.ink}"
    typography: "{typography.caption-strong}"
    rounded: "{rounded.pill}"
    padding: 4px 12px
  asset-icon-circular:
    backgroundColor: "{colors.surface-strong}"
    rounded: "{rounded.full}"
    size: 32px
  footer-light:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.body}"
    typography: "{typography.body-sm}"
    padding: 64px 48px
  footer-link:
    backgroundColor: transparent
    textColor: "{colors.body}"
    typography: "{typography.body-sm}"
  legal-band:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.muted}"
    typography: "{typography.caption}"
---

## Overview

Coinbase reads like an institutional financial brand that happens to trade crypto — the marketing surfaces are quiet, white-canvas, editorially-spaced, and almost monochromatic. The single brand voltage is **Coinbase Blue** (`#0052ff`), used scarcely: every primary CTA pill, the brand wordmark, and inline emphasis links. Beyond that one blue, the system is white canvas + ink + soft gray elevation bands + a deep near-black editorial canvas (`#0a0b0d`) for full-bleed product-mockup heroes.

Type pairs **CoinbaseDisplay** for hero headlines with **CoinbaseSans** for body, captions, and navigation. Display sits at **weight 400** — not the 700+ typical of trading platforms. The choice signals editorial calm and institutional trust rather than fintech urgency.

The page rhythm rotates three modes: bright white editorial sections, soft-gray elevation bands, and **full-bleed dark editorial heroes** carrying layered product-UI mockup cards. The dark hero with floating dashboard mockups is the single most distinctive component.

**Key Characteristics:**
- Single accent color: Coinbase Blue (#0052ff) carries every primary CTA, wordmark, and inline brand link. Used scarcely.
- Modest display weights — CoinbaseDisplay at weight 400, never 700+.
- Editorial pill geometry: every CTA is pill-shaped (100px radius), every asset glyph is full circle, every card is 24px radius. Sharp corners absent.
- Full-bleed dark heroes with floating product-UI cards is the brand's strongest signature pattern.
- Trading semantics: green (#05b169) and red (#cf202f) — text color only, never background fills.
- 96px section rhythm — generous editorial pacing.

## Colors

### Brand & Accent
- **Coinbase Blue** (#0052ff): The single brand color. Every primary CTA pill, the wordmark, and inline brand links.
- **Coinbase Blue Active** (#003ecc): Press-state darken on the primary pill.
- **Coinbase Blue Disabled** (#a8b8cc): Faded-blue tint for disabled CTAs.
- **Accent Yellow** (#f4b000): Used very sparingly on Bitcoin/asset glyph fills. Illustrative-only, not an action color.

### Surface
- **Canvas** (#ffffff): The default page floor.
- **Surface Soft** (#f7f7f7): Subtle alternating band surface.
- **Surface Strong** (#eef0f3): Light-gray fill behind secondary buttons, search pills, asset-icon plates.
- **Surface Dark** (#0a0b0d): Deep near-black canvas for full-bleed dark heroes, CTA bands.
- **Surface Dark Elevated** (#16181c): One step lighter, for floating product-UI mockup cards inside dark heroes.

### Text
- **Ink** (#0a0b0d): Display headings, primary nav, body emphasis.
- **Body** (#5b616e): Default running-text — slightly cool gray.
- **Body Strong** (#0a0b0d): Same as ink, for stronger emphasis.
- **Muted** (#7c828a): Sub-titles, breadcrumbs, footer secondary.
- **Muted Soft** (#a8acb3): Disabled link text.
- **On Primary** (#ffffff): White text on Coinbase Blue CTAs.
- **On Dark** (#ffffff): White text on dark heroes.
- **On Dark Soft** (#a8acb3): Muted off-white for secondary text on dark.

### Trading Semantics
- **Semantic Up** (#05b169): "Price up" green, text color only.
- **Semantic Down** (#cf202f): "Price down" red, text color only.

## Typography

### Font Substitutes
CoinbaseDisplay, CoinbaseSans, and CoinbaseMono are licensed typefaces. For BBGO SaaS:
- **CoinbaseDisplay → Inter** at weight 400, letter-spacing -1.5%
- **CoinbaseSans → Inter** at weight 400/600
- **CoinbaseMono → JetBrains Mono** or **Geist Mono** at weight 500

### Hierarchy

| Token | Size | Weight | Use |
|---|---|---|---|
| display-mega | 80px | 400 | Homepage hero h1 |
| display-xl | 64px | 400 | Subsidiary heroes |
| display-lg | 52px | 400 | Section heads |
| display-md | 44px | 400 | CTA-band headlines |
| display-sm | 36px | 400 | Sub-section heads |
| title-lg | 32px | 400 | Card group titles |
| title-md | 18px | 600 | Component titles, asset row primary |
| title-sm | 16px | 600 | List labels |
| body-md | 16px | 400 | Default body |
| body-strong | 16px | 700 | Emphasized body |
| body-sm | 14px | 400 | Footer body |
| caption | 13px | 400 | Photo captions |
| caption-strong | 12px | 600 | Badge pill labels |
| number-display | 18px | 500 | Asset prices, percent changes — monospace |
| button | 16px | 600 | Standard CTA pill |
| nav-link | 14px | 500 | Top-nav menu items |

### Principles
- **Display weight stays at 400.** Signals "calm institutional brand" rather than "trading-platform urgency."
- **Negative letter-spacing on display only.** Display uses -1px to -2px tracking; body stays at 0.
- **Monospace on every number.** Asset prices, percent changes — anything tabular renders in monospace.

## Layout

### Spacing System
- **Base unit:** 4px
- **Scale:** 4 · 8 · 12 · 16 · 20 · 24 · 32 · 48 · 96px
- **Section padding:** 96px for every major editorial band
- **Card internal padding:** 32px for feature cards and product-UI mockups

### Grid & Container
- **Max content width:** ~1200px centered
- **Feature card grids:** 2-up for hero splits, 3-up for benefit grids
- **Footer:** 6-column link list at desktop

## Elevation & Depth

| Level | Treatment | Use |
|---|---|---|
| Flat | No shadow, no border | 80% of surfaces |
| Hairline | 1px #dee1e6 | Feature card outlines on white |
| Soft drop | `0 4px 12px rgba(0,0,0,0.04)` | Hovered cards (single shadow tier) |

## Shapes

| Token | Value | Use |
|---|---|---|
| xs | 4px | Inline tags |
| sm | 8px | Compact rows |
| md | 12px | Form inputs |
| lg | 16px | Mid-size cards |
| xl | 24px | Feature cards, product-UI mockups, pricing tiers |
| pill | 100px | All CTA buttons, search pills, badges |
| full | 9999px | Asset icon circles, avatars |

## Do's and Don'ts

### Do
- Reserve Coinbase Blue for primary CTAs, wordmark, inline accent links.
- Set every CTA as pill-shaped (100px radius); every asset glyph as full circle.
- Keep display headlines at weight 400.
- Use dark/light band rotation as page rhythm.
- Render every numerical value in monospace.
- Pair every dark hero with a layered product-UI mockup card stack.

### Don't
- Don't introduce a secondary brand color. Coinbase Blue is the only action color.
- Don't bold display copy — display sits at weight 400.
- Don't add drop shadow tiers — system has one shadow tier.
- Don't use sharp corners on CTAs.
- Don't use trading green/red as a button background.

## Responsive Behavior

| Breakpoint | Width | Key Changes |
|---|---|---|
| Mobile | < 640px | Hero h1 80→40px; card grid 1-up; nav collapses to hamburger |
| Tablet | 640–1024px | Hero h1 64px; card grid 2-up |
| Desktop | 1024–1280px | Full hero h1 80px; card grid 3-up |
| Wide | > 1280px | Content caps at 1200px |

### Touch Targets
- Primary CTA pill at 44px height (WCAG AAA)
- Hero pill at 56px height
- Search pill at 44px height
