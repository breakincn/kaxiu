---
name: anchor-scroll-layout
description: Use when implementing or fixing in-page anchor scrolling for this project, especially mobile pages with sticky headers, grouped cards, inter-section spacing, and requirements like “scroll to the previous section border bottom plus 1-2px without exposing the previous border line”.
---

# Anchor Scroll Layout

Apply this skill when a page needs precise scroll landing behavior between stacked card groups.

Typical triggers:

- “跳转后滚动到某个分组”
- “不要露出上一分组底边线”
- “要保留当前分组和上一分组之间的间距”
- sticky header causes anchor misalignment

## Core rule

There are two different anchor targets. Do not mix them.

1. Scroll to an element top

- Use when the target section itself should贴着吸顶栏显示
- Formula: `element.top + window.scrollY - offsetPx`

2. Scroll to the previous section bottom plus a tiny gap

- Use when the design wants to show the inter-section spacing but not the previous section bottom border
- Formula: `previousSection.bottom + window.scrollY - stickyHeaderHeight + gapPx`
- Default `gapPx` is `2`

## Implementation pattern

- Prefer `ref` on the actual visual card container, not only the outer wrapper.
- Add shared helpers inside the page instead of duplicating ad-hoc `scrollTo` math:
  - `waitForScrollLayout()`
  - `getStickyHeaderHeight()`
  - `smoothScrollTo(top)`
  - `scrollElementToViewportTop(el, offsetPx)`
  - `scrollToAfterElementBottom(el, gapPx)`
- Wait for layout stability before scrolling:
  - `nextTick`
  - one `requestAnimationFrame`
  - short `setTimeout`

## Recommended defaults

- Sticky header height: query the actual `header`
- `gapPx`: `2`
- When scrolling to section top for regular anchors:
  - notice-like area: `4`
  - list/usages-like area: `12`

## Routing rule

- Trigger scroll by query string, for example:
  - `?scrollToAppointment=1`
  - `?scrollToUsages=1`
  - `?scrollToNotice=1`
- Perform the scroll only after the dependent data exists and the section has rendered.

## Review checklist

Search first:

```bash
rg -n "scrollTo|scrollIntoView|getBoundingClientRect|scrollTo[A-Z]|Anchor" frontend/src/views/user -S
```

Check:

- Is the `ref` attached to the correct visual box?
- Is the page using sticky header compensation?
- Is the design asking for section top alignment or previous-section-bottom alignment?
- Does the landing position avoid showing the previous section border line?
- Does it still preserve the intended gap between groups?

## Guardrails

- Do not anchor to an outer wrapper if the visual target is the inner card.
- Do not hardcode header height if it can be measured.
- Do not scroll before async data and conditional rendering finish.
- Do not use a single generic offset for all section types; choose based on layout intent.
