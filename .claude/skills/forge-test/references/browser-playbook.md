# Browser Playbook

Generic browser interaction patterns for QA testing.

## Setup

```
1. tabs_context_mcp → get available tabs
2. tabs_create_mcp → create a fresh tab
3. Store the tabId for all subsequent calls
```

## Login

```
1. navigate → {testUrl}/login
2. wait 2s
3. find → "Email input field" → triple_click → form_input "{username}"
4. find → "Password input field" → triple_click → form_input "{password}"
5. find → "Sign In button" → left_click
6. wait 3s → screenshot (verify dashboard)
```

Credentials come from `forge_config → get → previewDeploy.testCredentials`.

## Verify Element Visibility

**SHOULD be visible:** `find → "{description}"` → found = PASS, not found = FAIL + screenshot.

**Should NOT be visible:** `find → "{description}"` → found = FAIL, not found = PASS.

## Form Interaction

```
find → "{field}" → form_input value → find → "{submit}" → left_click → wait 2s → screenshot
```

## Override Browser Time

For day/time-dependent features:

```javascript
const __RealDate = window._RealDate || Date;
window._RealDate = __RealDate;
const __fakeTime = new __RealDate('2026-03-14T16:35:00').getTime();
const FakeDate = new Proxy(__RealDate, {
  construct(target, args) {
    if (args.length === 0) return new target(__fakeTime);
    return new target(...args);
  },
  get(target, prop) {
    if (prop === 'now') return () => __fakeTime;
    if (prop === 'prototype') return target.prototype;
    return target[prop];
  }
});
window.Date = FakeDate;
```

Override BEFORE triggering the action. Reload page to restore real time.

## Screenshots

Take at: after login, after navigation, at verification point, after action.

## General Rules

- Always `wait 2-3s` after navigation for SPA to load
- `triple_click` before `form_input` to clear pre-filled values
- Use `javascript_tool` with async IIFE (no top-level await)
