"""Check form field coverage - ensure frontend forms have all required fields."""

from pathlib import Path
from utils import CheckResult
from loaders import get_controller_validations, get_frontend_form_fields


def check_form_field_coverage(backend_dir: Path, frontend_dir: Path, result: CheckResult):
    """Check that frontend forms include all fields required by backend validation."""

    validations = get_controller_validations(backend_dir)
    form_fields = get_frontend_form_fields(frontend_dir)

    for resource, rules in validations.items():
        # Normalize resource name for matching
        normalized_resource = resource.replace('-', '')

        # Find matching frontend form
        matching_form = None
        for form_resource, form_data in form_fields.items():
            if form_resource.replace('-', '') == normalized_resource:
                matching_form = form_data
                break
            # Also try partial match
            if normalized_resource in form_resource.replace('-', '') or form_resource.replace('-', '') in normalized_resource:
                matching_form = form_data
                break

        if not matching_form:
            continue

        frontend_fields = matching_form['fields']
        conditionals = matching_form['conditionals']
        form_files = matching_form['files']

        for rule in rules:
            field = rule['field']

            if rule['type'] == 'required':
                # Simple required field - must be in form
                if field not in frontend_fields:
                    result.error(
                        'missing-form-field',
                        form_files[0] if form_files else f'frontend form for {resource}',
                        None,
                        f"Backend requires '{field}' but form doesn't have this field",
                        f"Add <Input> or <Select> for '{field}' in the form. Backend message: \"{rule['message']}\""
                    )

            elif rule['type'] == 'conditional':
                # Conditional required - field must be shown when condition is met
                condition_field = rule['condition_field']
                condition_value = rule['condition_value']

                if field not in frontend_fields:
                    result.error(
                        'missing-form-field',
                        form_files[0] if form_files else f'frontend form for {resource}',
                        None,
                        f"Backend requires '{field}' when {condition_field}='{condition_value}', but form doesn't have this field",
                        f"Add <Input> or <Select> for '{field}' shown when {rule['condition']}. Backend: \"{rule['message']}\""
                    )
                elif field in conditionals:
                    # Field exists - check if it's shown under the right condition
                    field_conditions = conditionals[field]
                    condition_met = any(
                        cf == condition_field and cv == condition_value
                        for cf, cv in field_conditions
                    )
                    if not condition_met:
                        result.warning(
                            'form-field-condition',
                            form_files[0] if form_files else f'frontend form for {resource}',
                            None,
                            f"Field '{field}' exists but may not be shown when {condition_field}='{condition_value}'",
                            f"Verify '{field}' is rendered when {rule['condition']}"
                        )
