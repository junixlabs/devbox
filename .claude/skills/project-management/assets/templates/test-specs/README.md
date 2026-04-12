# Test Specifications

Gherkin-format test specs derived from PRD features.

## Structure

```
test-specs/
├── README.md
└── features/
    └── {feature}.feature
```

## Gherkin Syntax

```gherkin
Feature: Feature name
  Description

  Scenario: Test case
    Given precondition
    When action
    Then result
```

## Feature Files

| Feature | PRD Source | Description |
|---------|------------|-------------|
| [{feature}.feature](features/{feature}.feature) | `prd/features/{feature}.md` | {Description} |

## Generate Test Specs

```bash
python3 scripts/add_test_spec.py <feature-name>
```
