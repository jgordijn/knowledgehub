## Context

The EntryCard component (`ui/src/lib/components/EntryCard.svelte`) renders entries in four visual tiers based on star rating: Featured (5★), High Priority (4★), Worth a Look (3★), and Low Priority (1-2★). Each tier has its own HTML template within a single `{#if}`/`{:else if}` chain.

**Current source display by tier:**

| Tier | Collapsed state | Expanded state |
|------|----------------|---------------|
| Featured (5★) | No source shown | Source avatar + name in bottom meta section |
| High Priority (4★) | No source shown | Source avatar + name in bottom meta section |
| Worth a Look (3★) | Source avatar only (no name) | Source avatar + name in bottom meta section |
| Low Priority (1-2★) | Source avatar only (no name) | Source avatar + name in bottom meta section |

The source info uses two helpers: `getSourceInitials()` returns a 2-char abbreviation, and `getSourceColor()` returns a deterministic color from a palette based on the source name hash. Both are already computed and available everywhere in the template.

## Goals / Non-Goals

**Goals:**
- Source name is always visible, even when a card is collapsed
- Source name appears near the title (top of card) rather than buried in the bottom meta section
- Maintain visual hierarchy — source should be secondary to the title, not competing with it
- Keep all four tier templates consistent in how they present source information

**Non-Goals:**
- Changing the source avatar color algorithm or size
- Making source name clickable/filterable (possible future feature)
- Changing card ordering or any other card content
- Modifying backend data or API responses

## Decisions

### 1. Source placement: directly below the title

**Decision**: Place the source avatar + name on a line immediately below the title, alongside the relative time. This replaces the current meta row at the bottom of the expanded section.

**Rationale**: Below the title is the natural scan position after reading the headline. Placing it there means it's visible whether collapsed or expanded without duplicating the element.

**Alternatives considered**:
- *Inline with title*: Would crowd the title line, especially on mobile where titles already truncate.
- *Keep in bottom meta, duplicate for collapsed*: Would mean maintaining source display in two places per tier — more code, more visual inconsistency risk.

### 2. Unified source line for all tiers

**Decision**: Use a consistent source + time line across all four tiers rather than tier-specific variations.

**Rationale**: The source line is informational, not a tier differentiator. Stars and card styling already communicate priority. Consistent placement makes scanning predictable.

### 3. Featured/HP collapsed state: add a compact header row

**Decision**: For Featured and High Priority tiers, when collapsed, show the title along with source + time beneath it (the same layout as when expanded, just without summary/actions). Currently these tiers show title + expand button only.

**Rationale**: Featured and HP currently show almost nothing when collapsed — just the title and a ★ label (Featured) or title (HP). Adding the source line gives enough context to decide whether to expand.

### 4. WaL/LP compact rows: add source name text

**Decision**: For the collapsed compact rows of Worth a Look and Low Priority, add the source name text next to the existing source avatar badge.

**Rationale**: The avatar initials alone are ambiguous when multiple sources share similar initials. Adding the short name makes identification instant. These rows already have the avatar, so the change is minimal.

## Risks / Trade-offs

- **[Horizontal space on mobile for WaL/LP rows]** → The compact rows already have stars, title, avatar, time, and expand button. Adding source name text could cause cramping on narrow screens. → **Mitigation**: Use `truncate` on the source name with a max-width, and let the title flex to fill remaining space. On very narrow screens the source name truncates gracefully.
- **[Visual weight shift]** → Moving source info higher might make cards feel busier at the top. → **Mitigation**: Keep the source line in small, muted text (11px, slate-400) so it remains secondary to the title.
