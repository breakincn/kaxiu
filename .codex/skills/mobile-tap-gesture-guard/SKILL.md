---
name: mobile-tap-gesture-guard
description: Use when fixing mobile pages where vertical scrolling accidentally triggers click or navigation, especially card/list rows that support tap-to-open, long-press menus, nested hot areas, and browser-synthesized click after touchend.
---

# Mobile Tap Gesture Guard

Apply this skill when a mobile list item can be tapped, long-pressed, or scrolled, and users are accidentally entering detail pages while trying to swipe vertically.

Typical triggers:

- “上下滑动时误触进入详情”
- “scroll 的时候触发了 click”
- “长按菜单和轻点跳转打架”
- “卡片里还有通知区/按钮区，滑动经过会误触”
- mobile Safari / H5 list row accidental navigation

Project example:

- See [references/project-examples.md](references/project-examples.md) when you want concrete examples from this repo, including the user card list page and the nested pinned-notice hot area.
- The same reference file also records two repo patterns for later rollout:
  - `CardDetail.vue`: partially aligned long-press list rows
  - `Dashboard.vue`: legacy mixed click + long-press pattern

## Core rule

Do not let raw `click` decide touch intent on mobile rows.

Use a controlled tap flow:

1. `touchstart`: record start position and active item id
2. `touchmove`: once movement crosses threshold, mark gesture as moved and cancel long press
3. `touchend`: only open detail when the gesture did not move and did not long-press
4. `click`: keep only as desktop/non-touch fallback, and suppress it after touch interactions

## Recommended state

Keep the state local to the page unless the same pattern already appears in many places:

- `startX/startY`
- `moved`
- `longPressed`
- `activeItemId`
- `suppressClickUntil`
- `longPressTimer`

## Recommended defaults

- move threshold: `8px`
- prefer vertical-scroll protection:
  - `abs(dy) >= 8 || abs(dx) >= 8`
  - or `abs(dy) >= 6 || dx*dx + dy*dy >= 8*8`
- short click suppression after scroll cancel: `350ms`
- click suppression after tap-open: `500ms`
- long press duration: `820ms`
- click suppression after long press: `900ms`

## Implementation pattern

For the main tappable row:

- `@click="onItemClick($event, id)"`
- `@touchstart="onItemTouchStart($event, item)"`
- `@touchmove="onItemTouchMove"`
- `@touchend="onItemTouchEnd($event, item)"`
- `@touchcancel="onItemTouchCancel"`

Behavior:

- `touchstart`
  - save touch coordinates immediately
  - set active visual press state
  - start long-press timer
- `touchmove`
  - if moved past threshold:
    - cancel long press
    - clear pressed visual state
    - set short `suppressClickUntil`
- `touchend`
  - if moved: only cleanup
  - if long-pressed: only cleanup
  - otherwise:
    - set `suppressClickUntil`
    - execute open/detail navigation directly from `touchend`
- `click`
  - if `Date.now() < suppressClickUntil`, return
  - otherwise handle desktop click

## Nested hot area rule

If the row contains a nested tappable zone, such as a pinned notice, CTA, or badge:

- do not use raw `@touchend="goToXxx()"` directly
- give the nested area its own guarded tap state
- stop propagation, but still apply the same move-threshold logic

Otherwise the nested area becomes the highest-risk accidental trigger during vertical scroll.

## Long-press rule

Long press should only do long-press work.

- It must not also fall through to detail navigation.
- When long press fires:
  - mark `longPressed = true`
  - suppress later click
  - clear pressed visual state
  - open the menu/sheet

## CSS assist

Add browser hinting, but do not rely on it as the main fix:

```css
.your-card-class {
  touch-action: pan-y;
}
```

This helps browsers prefer vertical panning, but it does not replace guarded `touchend` logic.

## Review checklist

Search first:

```bash
rg -n "@click|touchstart|touchmove|touchend|touchcancel|router.push\\(|goToDetail|suppressClickUntil|longPress" frontend/src -S
```

Check:

- Is `touchstart` recording coordinates immediately?
- Is `touchmove` able to cancel both long press and later click?
- Is detail open happening from guarded `touchend`, not from raw `click` alone?
- Is there a suppression window to absorb browser-synthesized click?
- Do nested hot areas have their own guarded tap logic?
- Does long press suppress later navigation?
- Has `active:scale` or similar visual feedback been removed if it makes scroll look like a click?

## Validation

Verify on a mobile simulator or real device:

1. tap once: open once
2. fast vertical swipe: do not open
3. slow drag past threshold: do not open
4. long press: open menu only
5. nested hot area tap: open intended target once
6. nested hot area swipe: do not open
7. after long press, releasing finger must not trigger a second navigation
8. desktop mouse click should still work

## Guardrails

- Do not keep raw `@touchend="navigate()"` on mobile hot areas.
- Do not record the initial touch point only on first `touchmove`.
- Do not rely on `click.prevent` alone to solve scroll misfires.
- Do not abstract into a shared composable unless at least several pages need the exact same pattern.
- When auditing a repo, classify pages first:
  - already aligned
  - partially aligned
  - legacy mixed model
  Then fix the highest accidental-navigation risk pages first.
