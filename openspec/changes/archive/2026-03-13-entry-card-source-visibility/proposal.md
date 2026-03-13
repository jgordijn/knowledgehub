## Why

The source (resource name) of an entry is currently buried in the meta section at the bottom of expanded cards, and for Featured/High Priority tiers it's completely invisible when collapsed. Users scanning their feed need to quickly identify where an article comes from — especially when collapsed — to decide whether to expand or skip it.

## What Changes

- **Move source display up**: Show the source name (with colored avatar) directly below the title on all card tiers, replacing the current placement in the bottom meta section.
- **Show source when collapsed**: For Featured and High Priority tiers (which currently hide the source when collapsed), add the source avatar/name to the collapsed header. Worth a Look and Low Priority already show the source avatar in collapsed rows — add the source name text next to it.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `feed-view`: The "Display entries as cards" requirement changes to specify that source name is always visible, including in collapsed state, and is positioned near the title rather than in the bottom meta area.

## Impact

- **Code**: `ui/src/lib/components/EntryCard.svelte` — restructure source display placement in all four tier templates (featured, hp, wal, lp)
- **No backend changes**: This is a pure UI layout change
- **No API changes**: The same `expand.resource.name` data is used, just displayed differently
- **No dependencies**: No new packages needed
