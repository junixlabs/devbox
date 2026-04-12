# Gherkin Guide

Quick reference for writing test specifications.

## File Structure

```
docs/requirements/test-specs/
├── README.md
└── features/
    ├── auth.feature
    ├── documents.feature
    └── {feature}.feature
```

## Syntax

```gherkin
Feature: Feature name
  Description of what this feature does

  Scenario: Single test case
    Given precondition is set up
    When user performs action
    Then expected result occurs
    And additional assertion

  Scenario Outline: Parameterized test
    Given I am "<role>"
    When I <action>
    Then I <result>

    Examples:
      | role   | action | result |
      | Admin  | edit   | can    |
      | Viewer | edit   | cannot |
```

## Keywords

| Keyword | Purpose | Example |
|---------|---------|---------|
| `Feature` | Group scenarios | `Feature: Login` |
| `Scenario` | Single test case | `Scenario: Valid login` |
| `Given` | Precondition | `Given I am logged in` |
| `When` | Action | `When I click submit` |
| `Then` | Assertion | `Then I see dashboard` |
| `And` / `But` | Continue step | `And I see welcome message` |
| `Scenario Outline` | Data-driven | With `Examples` table |
| `Examples` | Test data | Table of values |

## Best Practices

1. **One behavior per scenario** - Keep scenarios focused
2. **Use business language** - Avoid technical details
3. **Reusable steps** - Write steps that can be shared
4. **Declarative > Imperative** - Describe what, not how

## Converting PRD to Gherkin

**PRD Acceptance Criteria:**
```markdown
- [x] Admin can create users
- [x] Email validation required
- [x] Role assignment required
```

**Gherkin Scenarios:**
```gherkin
Scenario: Admin creates user
  Given I am logged in as "Admin"
  When I create user with email "test@example.com"
  And I assign role "Editor"
  Then user is created successfully

Scenario: Email validation
  Given I am creating a user
  When I enter invalid email "not-an-email"
  Then I see "Invalid email" error

Scenario: Role is required
  Given I am creating a user
  When I leave role empty
  Then I see "Role is required" error
```

## Generate from PRD

```bash
python3 scripts/add_test_spec.py <feature-name>
```

Creates template with PRD criteria as comments for reference.
