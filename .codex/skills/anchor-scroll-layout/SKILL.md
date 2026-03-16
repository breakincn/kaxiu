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
- “目标分组在页面底部，页面长度不够，滚不到指定锚点”
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

## Bottom-edge constraint

When the target section is near the page bottom, the computed anchor may be correct but still unreachable because the page has no remaining scroll space.

In that case, do not relax the anchor requirement. Instead:

1. compute the intended `targetScrollTop`
2. compute current max scroll: `documentScrollHeight - viewportHeight`
3. if `targetScrollTop` is larger than current max scroll, append dynamic bottom spacer
4. wait for layout again
5. then perform the scroll

Recommended spacer formula:

- `requiredSpacer = max(0, targetScrollTop - maxScrollTop + bufferPx)`
- default `bufferPx` can be `24`

## Implementation pattern

- Prefer `ref` on the actual visual card container, not only the outer wrapper.
- Add shared helpers inside the page instead of duplicating ad-hoc `scrollTo` math:
  - `waitForScrollLayout()`
  - `getStickyHeaderHeight()`
  - `smoothScrollTo(top)`
  - `getMaxScrollTop()`
  - `ensureScrollableSpaceFor(targetScrollTop, bufferPx)`
  - `scrollElementToViewportTop(el, offsetPx)`
  - `scrollToAfterElementBottom(el, gapPx)`
- Wait for layout stability before scrolling:
  - `nextTick`
  - one `requestAnimationFrame`
  - short `setTimeout`
- If the target is near the page bottom, expand bottom spacer before scrolling instead of accepting a degraded landing position.

## Recommended defaults

- Sticky header height: query the actual `header`
- `gapPx`: `2`
- bottom-edge buffer: `24`
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
- If the section is at the page bottom, has the code added enough dynamic bottom spacer to make the anchor reachable?

## Guardrails

- Do not anchor to an outer wrapper if the visual target is the inner card.
- Do not hardcode header height if it can be measured.
- Do not scroll before async data and conditional rendering finish.
- Do not use a single generic offset for all section types; choose based on layout intent.
- Do not accept “scroll as far as possible” as a fallback when the product explicitly requires a precise landing point near the page bottom.
