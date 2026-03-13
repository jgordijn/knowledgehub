## 1. Featured Tier (5★) — Source Visibility

- [x] 1.1 Add source avatar + name + relative time line below the title in the Featured tier, visible in both collapsed and expanded states (move from bottom meta section to below the `<h3>` title)
- [x] 1.2 Remove the source display from the bottom meta section in the Featured expanded area (keep stars display there, remove source avatar/name/time since it's now above)

## 2. High Priority Tier (4★) — Source Visibility

- [x] 2.1 Add source avatar + name + relative time line below the title in the HP tier, visible in both collapsed and expanded states (move from bottom meta section to below the `<h3>` title)
- [x] 2.2 Remove the source display from the bottom meta section in the HP expanded area (keep star rating widget, remove source avatar/name/time)

## 3. Worth a Look Tier (3★) — Source in Collapsed Row

- [x] 3.1 Add source name text next to the existing source avatar in the WaL compact collapsed row
- [x] 3.2 Move source avatar + name + relative time from the expanded detail panel's meta section to directly below the title in the expanded panel

## 4. Low Priority Tier (1-2★) — Source in Collapsed Row

- [x] 4.1 Add source name text next to the existing source avatar in the LP compact collapsed row
- [x] 4.2 Move source avatar + name + relative time from the expanded detail panel's meta section to directly below the title in the expanded panel

## 5. Verification

- [x] 5.1 Build the frontend (`cd ui && bun run build`) and verify no build errors
- [x] 5.2 Visually verify all four tiers show source below title when expanded
- [x] 5.3 Visually verify all four tiers show source when collapsed
