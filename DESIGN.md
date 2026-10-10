# Design — Project Identity

> This document is project-long-lived. Tokens are not changed without
> the Architect's approval. Developers MUST use these tokens
> instead of improvising their own colors/spacings.

## Style Direction

Calm, trustworthy workshop tool: near-white neutral surfaces, one deep "workshop blue" accent, restrained amber/green status signals, system fonts only — quiet like a banking app, not flashy like a car ad.

## Colors

- `--color-bg`: **#F6F7F9**
- `--color-surface`: **#FFFFFF**
- `--color-surface-sunken`: **#EEF0F3**
- `--color-fg`: **#1A1D21**
- `--color-fg-muted`: **#5B6470**
- `--color-fg-subtle`: **#858E9A**
- `--color-border`: **#DCE0E5**
- `--color-border-strong`: **#B9C0C9**
- `--color-accent`: **#1E4E8C**
- `--color-accent-hover`: **#17407A**
- `--color-accent-active`: **#123564**
- `--color-accent-soft`: **#EAF0F9**
- `--color-accent-contrast`: **#FFFFFF**
- `--color-focus-ring`: **#7BA7DE**
- `--color-status-requested-bg`: **#EEF0F3**
- `--color-status-requested-fg`: **#5B6470**
- `--color-status-confirmed-bg`: **#EAF0F9**
- `--color-status-confirmed-fg`: **#1E4E8C**
- `--color-status-inprogress-bg`: **#FCF3E3**
- `--color-status-inprogress-fg`: **#8A5A12**
- `--color-status-done-bg`: **#E9F4EE**
- `--color-status-done-fg`: **#22664A**
- `--color-status-pickedup-bg`: **#F0F1F3**
- `--color-status-pickedup-fg`: **#6B7280**
- `--color-success`: **#2F7D5B**
- `--color-warning`: **#B7791F**
- `--color-danger`: **#B42318**
- `--color-danger-soft`: **#FBEAE8**
- `--color-success-soft`: **#E9F4EE**
- `--color-info-soft`: **#EAF0F9**
- `--color-shadow-color`: **rgba(16, 24, 40, 0.08)**

## Typography

- `font_family`: system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, 'Noto Sans', sans-serif
- `font_family_mono`: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace
- `heading_weight`: 600
- `body_weight`: 400
- `label_weight`: 500
- `size_xs`: 12px
- `size_sm`: 13px
- `size_base`: 15px
- `size_lg`: 17px
- `size_h3`: 20px
- `size_h2`: 24px
- `size_h1`: 30px
- `line_height_body`: 1.55
- `line_height_heading`: 1.25
- `letter_spacing_heading`: -0.01em
- `numeric_style`: font-variant-numeric: tabular-nums for all amounts, kilometre readings, dates and order numbers

## Spacing Scale

- `--space-0`: 4px
- `--space-1`: 8px
- `--space-2`: 12px
- `--space-3`: 16px
- `--space-4`: 24px
- `--space-5`: 32px
- `--space-6`: 48px

## Border-Radii

- `--radius-sm`: 4px
- `--radius-md`: 8px
- `--radius-lg`: 12px
- `--radius-pill`: 999px

## Components

### Button

Variants: primary (bg=accent #1E4E8C, text=#FFFFFF), secondary (bg=surface #FFFFFF, text=fg #1A1D21, 1px border #DCE0E5), danger (bg=#B42318, text=#FFFFFF, only for destructive actions), ghost (transparent bg, text=accent, no border, used in nav and table row actions). Geometry: padding 12px 20px, radius md (8px), min-height 44px and min-width 44px on all viewports (tap target), font 15px/600, letter-spacing 0, optional 16px leading icon at 1.5px stroke. States: default; hover primary bg=#17407A + translateY(-1px) + box-shadow 0 1px 2px rgba(16,24,40,.08), hover secondary bg=#F6F7F9 + border #B9C0C9; active bg=#123564 (secondary #EEF0F3), translateY(0), box-shadow none; focus-visible outline 2px #7BA7DE, outline-offset 2px (never outline:none); disabled opacity 0.5, cursor not-allowed, no hover/active change, aria-disabled and native disabled; loading state swaps the label for a 16px spinner and sets aria-busy=true. Full-width (width:100%) on viewports <640px inside forms.

### Input & Textarea

Height 44px (textarea min-height 96px, resize vertical only), padding 10px 12px, bg=surface, 1px border #DCE0E5, radius md 8px, font 15px, text fg, placeholder #858E9A. Label above the field: 13px/500, fg-muted, 8px gap, always visible (no placeholder-as-label). Focus: border #1E4E8C + outline 2px #7BA7DE (offset -1px), no shadow change. Error: border #B42318, background #FBEAE8 at 40% opacity, helper text 13px #B42318 below with error machine code shown in title attribute. Disabled: bg #EEF0F3, text #858E9A. Field-specific: licence plate input uses font-family_mono, uppercase transformation and placeholder 'B-AB 1234'; kilometre and amount inputs use tabular-nums and inputmode='numeric'; date inputs use a native <input type='date'> and are never re-formatted in the field.

### Card

bg=surface #FFFFFF, 1px border #DCE0E5, radius lg 12px, padding 24px (16px below 640px), box-shadow only when elevated: 0 1px 2px rgba(16,24,40,.08). Structure: CardHeader (title 20px/600 + optional right-aligned action), CardBody, CardFooter (top border #DCE0E5, padding-top 16px, actions right-aligned on desktop / stacked full-width below 640px). Used for: order detail, invoice summary, dashboard tiles, login form. Section titles inside a card are 17px/600, never larger than the page title.

### StatusBadge

Pill (radius 999px), padding 4px 10px, font 13px/600, letter-spacing 0.01em, no icon by default, optional 6px dot in the fg colour. Fixed mapping of the five lifecycle states — never invent new colours: angefragt = bg #EEF0F3 / fg #5B6470; bestätigt = bg #EAF0F9 / fg #1E4E8C; in Arbeit = bg #FCF3E3 / fg #8A5A12; fertig = bg #E9F4EE / fg #22664A; abgeholt = bg #F0F1F3 / fg #6B7280. Badge text is the German state name exactly as the API returns it. Contrast ratio ≥ 4.5:1 for every pair; never rely on colour alone — the state name is always written out, also in the mobile table view.

### DataTable

Workshop order list. Desktop ≥900px: real table, header row 13px/600 fg-muted uppercase letter-spacing 0.04em on bg #EEF0F3, rows 52px min-height, 1px bottom border #DCE0E5, hover row bg #F6F7F9, numeric/fixed columns (order number, plate, date) tabular-nums and right- or centre-aligned, order number and plate in font_family_mono. Mobile <900px: rows collapse into stacked cards (20px gap) with label-above-value pairs at 13px/500 fg-muted for labels; the StatusBadge stays visible without scrolling horizontally. No horizontal scrolling of the page at any width. Row actions (bestätigen, Positionen erfassen, Status setzen) live in a trailing cell/area and additionally exist in the row's detail view — never icon-only without aria-label.

### FilterBar

Above the order list. Left: status filter as a segmented row of toggle chips (radius pill, height 36px, padding 6px 14px, font 13px/600) with an 'Alle' chip plus one chip per lifecycle state carrying the StatusBadge colour when active (inactive: bg surface, border #DCE0E5, fg fg-muted). Chips wrap onto multiple lines below 640px instead of scrolling. Right: search input for the licence plate (min-width 220px, placeholder 'Kennzeichen suchen', mono font), 24px gap between the two groups, filters placed directly above the table with 16px spacing. Active filter count is announced via aria-live; an empty result uses the EmptyState component.

### StatTile

Dashboard row, three tiles: open orders, orders finished today, revenue of the current month. bg=surface, radius lg 12px, border #DCE0E5, padding 24px, label 13px/500 fg-muted uppercase letter-spacing 0.04em, value 30px/600 fg tabular-nums with 8px gap above, optional delta line 13px in success/danger. Layout: 3 columns ≥1200px, 2 columns 900–1199px (revenue tile spans the full second row), 1 column below 640px — never a cramped 3-across on mobile. Revenue value always rendered by the shared money formatter.

### AppShell & Navigation

Top bar height 64px (56px below 640px), bg=surface, 1px bottom border #DCE0E5, sticky, z-index 10. Left: wordmark 'Werkstattportal' 17px/600 plus a small written-out section label ('Kundenbereich' / 'Werkstattbereich') in fg-muted; right: nav links (15px/500, active link accent with 2px bottom border) and, in the workshop area, the signed-in employee name with a sign-out ghost button. Below 900px the links collapse into a disclosure menu opened by a 44x44px button labelled 'Menü' (never a bare hamburger without accessible name). Content area: max-width 1200px, centred, 24px horizontal padding (16px below 640px), 32px vertical spacing between sections. Both areas render the same footer with links to Impressum and Datenschutzerklärung, present and reachable from every page.

### Form & Validation

Vertical stack with 20px between fields, 24px before the submit row. Every field has a visible label, optional 'erforderlich' marker, and a reserved helper-text slot (13px, 20px line-height) so that error messages never shift the layout. Client validation mirrors size and format (email, phone, required, integer kilometre reading, euro amount with comma); server errors from the shared API error format are mapped to the field whose machine code matches and otherwise shown as a form-level Alert. Error summary at the top of the form with an anchor per field for keyboard users. Submit button is disabled while the request is in flight and shows the loading spinner; a successful submit replaces the form with a confirmation panel showing the assigned order number in mono type.

### Alert & Toast

Inline Alert: radius md 8px, padding 12px 16px, 1px border, 15px text, tones info (#EAF0F9 / #1E4E8C), success (#E9F4EE / #2F7D5B), warning (#FCF3E3 / #8A5A12), error (#FBEAE8 / #B42318), each with a left 3px accent bar and a 16px icon; used for form-level errors and status feedback, placed 16px above the affected block. Toast: bottom-centre on mobile, bottom-right on desktop, min 320px, radius md, 8px shadow, auto-dismiss after 6s, text is always a sentence a workshop customer understands — never the raw machine code alone. Role='status' for success, role='alert' for errors.

### Modal / Dialog

Radius lg 12px, padding 24px, max-width 480px (form dialogs) / 720px (position editor), bg surface, overlay rgba(16,24,40,0.45), entrance fade 120ms ease-out, no bounce. Focus is trapped inside, ESC closes, click on the overlay closes, and focus returns to the trigger element. Header 20px/600 with a 44x44px close button labelled 'Schließen'; footer actions right-aligned, primary rightmost, stacked full-width below 640px. Used for 'Positionen erfassen' and status changes with confirmation text naming the order number and the target state.

### EmptyState

Centred block inside a Card, max-width 420px, vertical padding 48px: 48px outline illustration (inline SVG, accent-soft stroke), title 17px/600, one explanatory sentence 15px fg-muted, optional secondary action. Used for 'no orders match these filters', 'no status events yet' and 'no invoice yet, it appears once the order is finished'. Never an apologetic or technical message; always tell the user the next useful step.

### StatusTimeline

Order status log (AC-06) in the customer and workshop order view. Vertical list, 24px left padding, 2px #DCE0E5 connector line, 10px node per event filled with the StatusBadge fg colour of the new state, most recent event first but reading bottom-to-top chronologically is avoided — order is newest at the top with an explicit heading. Each entry: state name 15px/600, timestamp 13px fg-muted in the shared date-time format, optional actor line. Unreached future states are rendered as outlined nodes in fg-subtle with 'ausstehend'. Never show a state that was not actually reached.

### InvoiceSummary

Customer invoice view (AC-12). Line items table: description (Arbeitszeit / Einzelteil with quantity), quantity, unit price, line total — all amounts right-aligned, tabular-nums, mono for the order number. Then a totals block on bg #F6F7F9, radius md 8px, padding 16px: Nettobetrag, Mehrwertsteuer (19 %), Gesamtbetrag 20px/600 with a 1px #DCE0E5 separator above it. Every amount is rendered from integer cents by the shared formatter — the frontend never does float arithmetic for display. No download or payment button is shown (out of scope): instead a plain note that the invoice is visible here only.

### LegalFooter

Present in both areas on every page: 1px top border #DCE0E5, padding 24px, font 13px fg-muted, links 'Impressum' and 'Datenschutzerklärung' with 16px gap, plus a line 'Alle Angaben ohne Gewähr. Werkstattportal, Beispielwerkstatt'. Links use accent on hover with an underline; below 640px they stack vertically with 8px gaps. No third-party assets, no external font or script references anywhere in the product (data-protection requirement).

## Layout Principles

- Container: max-width 1200px, centred, 24px horizontal padding (16px below 640px); reading-heavy blocks (forms, invoice summary, legal pages) are additionally capped at 720px.
- Breakpoints (mobile-first, min-width): 640px (small tablet, buttons stop stretching full width), 900px (tablet landscape, table becomes a real DataTable, nav collapses/stays), 1200px (desktop, dashboard tiles go 3-across). Nothing is designed for a width below 360px but nothing breaks there either.
- Layout system: CSS Grid for page-level rows (dashboard tiles, form/result split) and flex for inline groups; 12-column grid on desktop for the workshop area, single column on mobile. No fixed pixel widths on content wrappers, no horizontal scrolling at any width.
- Vertical rhythm: 32px between page sections, 24px between cards, 16px between a filter bar and its table, 8px between a label and its field, 20px between form fields. Page top padding 32px (24px below 640px).
- Order within a page: one h1 (30px/600) that names the task, optional one-line fg-muted subtitle, then content; section headings 20px/600, card headings 17px/600 — never skip or duplicate the h1.
- Elevation: flat by default (borders, not shadows). Shadow only for sticky nav on scroll, toasts and modals. One accent colour per screen — accent is reserved for the primary action and focus, never for decoration.
- Interaction: every interactive element has min 44x44px touch target, a visible focus ring (2px #7BA7DE, offset 2px), hover feedback on pointer devices only, and an accessible name. Anything not yet functional is rendered visibly disabled and labelled 'kommt bald' (AC-19) — never a dead-looking control that appears active.
- Responsiveness discipline: text never smaller than 13px, tap targets never shrink below 44px, no truncation of licence plates, amounts or order numbers (they wrap instead), and long problem descriptions wrap with overflow-wrap:anywhere inside their card.
- ONE canonical formatting for every value in both areas, implemented once and reused (AC-20/AC-12): date = DD.MM.YYYY (05.03.2025); date + time = DD.MM.YYYY, HH:MM Uhr; time only = HH:MM Uhr; weekday-relative labels only as an extra word, never instead of the date; duration = German hours/minutes, e.g. '2 h 30 min' (never '2.5 h'); money = integer cents rendered as German currency, thousands dot and comma decimals with two decimals and a non-breaking space before €, e.g. '1.234,56 €' (negative '-1.234,56 €'); kilometre reading = thousands-separated plus ' km', e.g. '128.450 km'; VAT always written as 'Mehrwertsteuer (19 %)'; order number = mono, uppercase, pattern 'AU-JJJJ-NNNNN', e.g. 'AU-2025-00123'; licence plate = mono, uppercase, normalised as 'B-AB 1234'. Dates and amounts use tabular-nums everywhere so columns line up.
- Accessibility baseline: WCAG AA contrast on all text (fg #1A1D21 on #F6F7F9 and #FFFFFF, accent #1E4E8C on white for links), status never conveyed by colour alone, forms fully keyboard-operable with labels and error associations, aria-live for toasts and filter result counts, and a logical heading order per page.
