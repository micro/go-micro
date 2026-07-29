# DESIGN.md — Landing page style system

Style contract for the Hugo/ Docsy landing pages (`content/en/_index.md`,
`content/en/support.md`) and the shortcodes that build them
(`layouts/_shortcodes/blocks/*`, `layouts/_shortcodes/elements/*`).

## Color tokens

Defined in `assets/scss/_variables_project_after_bs.scss` and merged into
Bootstrap's `$theme-colors`, so Docsy generates a `td-box--{color}` modifier
for each:

| token         | hex       | use                                  |
|---------------|-----------|--------------------------------------|
| `dark-blue`   | `#111827` | primary dark section background      |
| `deep-blue`   | `#03045e` |                                      |
| `teal-blue`   | `#0077b6` |                                      |
| `turquoise`   | `#00b4d8` | brand cyan                           |
| `frosted-blue`| `#90e0ef` |                                      |
| `light-cyan`  | `#caf0f8` |                                      |

Plus Docsy built-ins: `primary`, `secondary`, `dark`, `light`, `white`, `gray`.

A section background is set with `color="<token>"`, which renders
`td-box--<token>`. `td-box--dark-blue` and `td-box--light` are the two used on
the landing pages.

## `bg-pattern`

`assets/scss/_variables_project_after_bs.scss` → a radial dotted overlay
(`rgba(0,173,216,0.15)` dots on a 20px grid). Applied as a modifier class on a
section or hero. Toggle it with the shortcode `pattern` param (see below), not
by hand.

## Section rhythm

The landing pages (`content/en/_index.md`, `content/en/support.md`) are
**uniformly dark**: every `blocks/section` uses `color="dark-blue"
pattern=true padding="py-5"`. Do **not** introduce `light` sections — the site
is dark-mode-forced, so a light section would clash and was deliberately removed.
Adjacent sections share the same background by design; content, headings, and
the `bg-pattern` texture carry the separation.

When a section holds dark-theme syntax-highlighted `code` (GitHub dark palette:
`#ff7b72`/`#d2a8ff`/`#a5d6ff`), keep it `dark-blue` so the snippet reads.

## Shortcode contracts

### `blocks/hero`

```go-html-template
{{% blocks/hero
  height="max"          /* auto | min | med | max | full */
  color="dark-blue"     /* td-box-- color token */
  pattern=true          /* optional: add bg-pattern overlay */
%}}
...inner content...
{{% /blocks/hero %}}
```

> Legacy: `td-below-navbar` is still appended to `height` as a navbar-offset
> class (`height="max td-below-navbar"`). Move it to a dedicated param when one
> is added — do not introduce new classes via `height`.

### `blocks/section`

```go-html-template
{{% blocks/section
  color="dark-blue"     /* td-box-- color token; defaults to auto-alternating by ordinal */
  height="auto"         /* auto | min | med | max | full */
  type="row"            /* container | row | text-center | ... Bootstrap utilities */
  pattern=true          /* optional: add bg-pattern overlay */
  padding="py-5"        /* optional: vertical padding utility; omit to use type/legacy */
%}}
...inner content...
{{% /blocks/section %}}
```

### `blocks/link-down`

```go-html-template
{{% blocks/link-down color="info" %}}
```

### `elements/variant-card`

```go-html-template
{{% elements/variant-card
  color="gradient"      /* dark | light | gradient */
  title="Pluggable"
  subtitle="Swap components without changing code"
  content="top"         /* top | bottom: where .Inner (icon/preview) sits */
%}}
<div class="p-3 display-6">🔌</div>
{{% /elements/variant-card %}}
```

## Anti-pattern (legacy — being retired)

Do **not** pack extra classes into a single param:

```go-html-template
{{% blocks/section color="dark-blue bg-pattern py-5" type="row" %}}   <!-- WRONG -->
{{% blocks/section color="dark-blue" pattern=true padding="py-5" type="row" %}}  <!-- RIGHT -->
```

`color`, `height`, and `type` are single-purpose. `pattern` and `padding` are
first-class; alignment utilities (`text-center`) belong in `type`.

`content/en/about/index.md` still uses the legacy `type="text-center h1 py-4"`
(alignment + padding in `type`). Migrate it to `type="text-center"`
+ `padding="py-4"` when touched.

## Sponsor logos

In the `_index.md` Sponsors section, each logo is a local `<img>` forced white
with a CSS filter (the section background is `dark-blue`):

```html
<a href="/blog/2026/03/04/building-the-ai-native-future-of-go-micro-with-claude/"><img src="/images/sponsors/anthropic.svg" alt="Anthropic" class="sponsor-logo" /></a>
```

`.sponsor-logo` applies `filter: brightness(0) invert(1)` — pure white
regardless of the SVG's own fill. Using `<img>` (not a CSS mask) keeps the
intrinsic size, so the logo can't collapse to zero width inside the `d-flex`
row. Assets live in `static/images/sponsors/`. Swap a logo by replacing the
file and the `src`; the filter handles the color.
