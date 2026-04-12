# Test Approach

## Think Like a Human Tester

Read the issue from a user's perspective — what should work differently after this change? Don't test implementation details. Test observable behavior by following the user journey: navigate to the feature, perform the action, verify the result.

## Deriving Test Cases from Pipeline Context

### From Acceptance Criteria (primary)
Each acceptance criterion becomes at least one test case:
- "User can X" → navigate to the feature and do X
- "Y should display Z" → navigate to Y and verify Z is visible
- "API returns X when Y" → call API with Y and check response

### From Plan (secondary)
The plan field often contains testable edge cases:
- Per-role behavior → test with multiple credential roles
- Day/schedule overrides → test for specific days/conditions
- Fallback logic → test when primary path is unavailable

### From Review Comments (tertiary)
If forge-review posted findings, check for flagged edge cases.

### From Previous QA Failures (reopen cycles)
If changeHistory shows a prior `testing → reopen` transition:
1. Find the most recent "QA Test Report" comment
2. Extract all FAIL items — these are mandatory regression tests
3. Tag them as `Regression` source in the new report

## Backend API Testing

Use `testApiUrl` as the base URL for all API calls.

**Authentication:**
```bash
curl -s -X POST "$TEST_API_URL/api/auth/local" \
  -H "Content-Type: application/json" \
  -d '{"identifier":"user@example.com","password":"..."}' | jq -r '.jwt'
```

**What to verify:**
- Correct HTTP status codes (200, 201, 400, 404)
- Response body shape matches expected schema
- Data values are correct (not just present)
- Edge cases from acceptance criteria

## Frontend UI Testing

Use `testUrl` as the base URL.

**With browser tools (preferred):**
- Navigate to pages, read content, find elements, fill forms, take screenshots

**Without browser tools (fallback):**
- Use `curl -s "$TEST_URL/path"` to fetch page HTML
- Check for key element text or data attributes
- Note in report: "Tested via HTML inspection"

**Visual check:** Look for broken layouts, overlapping elements, missing styles, console errors, blank sections, misaligned content, or any UI corruption. Report these as FAIL even if the feature works functionally.
