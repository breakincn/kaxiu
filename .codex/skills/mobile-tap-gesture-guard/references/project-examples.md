# Project Examples

This file gives concrete examples from the current repo for the `mobile-tap-gesture-guard` skill.

## User card list

Primary reference:

- [frontend/src/views/user/CardList.vue](/Users/will/Projects/Go/kabao/frontend/src/views/user/CardList.vue)

What changed there:

- card row uses guarded tap handlers instead of raw touch-to-click fallthrough
- long press opens the action sheet without later triggering detail navigation
- nested pinned notice uses its own guarded touch state instead of raw `touchend` navigation
- the page keeps desktop click behavior through `suppressClickUntil`

Useful search targets:

```bash
rg -n "onCardTouchStart|onCardTouchMove|onCardTouchEnd|onNoticeTouchStart|onNoticeTouchEnd|suppressClickUntil" frontend/src/views/user/CardList.vue -S
```

## CSS assist

Reference:

- [frontend/src/style.css](/Users/will/Projects/Go/kabao/frontend/src/style.css)

Useful search target:

```bash
rg -n "touch-action: pan-y|\\.kb-card" frontend/src/style.css -S
```

## Similar follow-up targets in this repo

These are good candidates when expanding the same guard pattern later:

- [frontend/src/views/user/CardDetail.vue](/Users/will/Projects/Go/kabao/frontend/src/views/user/CardDetail.vue)
  - usage rows already use long-press timing and movement thresholds
  - this file is the “partial alignment” example in this repo:
    - good: records touch start immediately
    - good: cancels long press after movement
    - review later if usage rows also need desktop-click fallback or stricter guarded tap consistency
- [frontend/src/views/merchant/Dashboard.vue](/Users/will/Projects/Go/kabao/frontend/src/views/merchant/Dashboard.vue)
  - top scan entry is the “legacy mixed model” example in this repo:
    - click still decides the normal tap path
    - long press uses delayed suppression
    - touch start point is still initialized lazily on first move
  - this is a strong candidate when applying the same controlled-tap pattern outside the user card list

Suggested repo-wide search:

```bash
rg -n "@touchstart|@touchmove|@touchend|suppressClickUntil|longPressTimer|router.push\\(" frontend/src/views -S
```

## Suggested rollout order

When expanding the skill in this repo, use this order:

1. user-facing accidental navigation pages first
   - pages where vertical swipe can unexpectedly enter a detail page
2. nested hot-area pages second
   - list rows that contain pinned notices, CTA chips, or badges
3. merchant tools last
   - places like top scan entries where long press is intentional but accidental navigation is less frequent
