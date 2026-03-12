---
name: merchant-terms
description: Use when working on this project’s service terminology, especially any UI, API, copy, status text, alerts, or business rules involving 起单/结单/叫号/结号. Enforces the project rule: queue mode maps default 起单->叫号 and 结单->结号; customer-service mode maps 起单/结单 to merchant-configured start_term/finish_term from 商家信息设置.
---

# Merchant Terms

Apply this skill whenever a task touches service lifecycle text, labels, button copy, status copy, prompts, alerts, comments that affect product behavior, or API responses involving:

- `起单`
- `结单`
- `叫号`
- `结号`
- `上号`
- any derived phrases such as `待起单`, `起单成功`, `结单后暂停`, `今日起单记录`

## Canonical rule

There are only two term-resolution modes:

1. Queue mode

- Default `起单` becomes `叫号`
- Default `结单` becomes `结号`

2. Customer-service mode

- `起单` must use merchant `start_term`
- `结单` must use merchant `finish_term`
- The values come from `商家信息设置 -> 起单/结单设置`

If neither custom value exists in customer-service mode, fall back to default `起单` / `结单`.

`上号` is not a blanket replacement for `起单`.
Use `上号` only for the explicit scan/start step that is truly “scan to put on service”, not as the generic global replacement term.

## Implementation rules

- Frontend must prefer shared helpers in [frontend/src/utils/terms.js](../../../../frontend/src/utils/terms.js).
- Backend must prefer shared helpers in `backend/handlers` term helpers instead of open-coding `start_term` / `finish_term` logic.
- Do not hardcode `起单` / `结单` in user-facing strings if the text should follow merchant mode rules.
- When adding new pages or APIs, first decide whether the text is:
  - generic lifecycle wording: route through shared term helpers
  - explicit queue wording: write `叫号` / `结号`
  - explicit scan wording: write `扫码上号`

## Review checklist

When changing code, search for:

```bash
rg -n "起单|结单|叫号|结号|上号|start_term|finish_term|replaceTerms|getPendingStartLabel" frontend backend -S
```

Check each hit:

- Is it user-facing?
- Is it generic lifecycle wording?
- Is it queue-mode specific?
- Should it use shared helpers instead of hardcoded text?

## Guardrails

- Do not replace explicit `扫码上号` with `扫码叫号`.
- Do not replace queue-progress concepts that are intentionally `待上号` unless the product requirement explicitly changes that concept.
- Do not add another ad-hoc term helper in a page/component/handler if a shared helper can own the rule.

