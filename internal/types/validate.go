package types

import (
	"fmt"

	"github.com/HJyup/patchdock/internal/validate"
	"github.com/go-playground/validator/v10"
)

var contractValidator = newContractValidator()

func newContractValidator() *validate.Validator {
	v := validate.New("json", map[string]validate.Translator{
		"min": func(path string, _ validator.FieldError) error {
			return fmt.Errorf("%s: empty", path)
		},
		"duplicate": func(path string, fieldErr validator.FieldError) error {
			return fmt.Errorf("%s: duplicate of %s", path, fieldErr.Param())
		},
		"patch_required": func(path string, fieldErr validator.FieldError) error {
			return fmt.Errorf("%s: empty for %s status", path, fieldErr.Param())
		},
		"issues_required": func(path string, _ validator.FieldError) error {
			return fmt.Errorf("%s: required when decision is reject", path)
		},
		"issues_forbidden": func(path string, _ validator.FieldError) error {
			return fmt.Errorf("%s: must be empty when decision is accept", path)
		},
	})

	v.RegisterStructValidation(validatePlan, Plan{})
	v.RegisterStructValidation(validateReview, Review{})
	return v
}

func validateStruct(value any, root string) error {
	return contractValidator.Struct(value, root)
}

func validatePlan(sl validator.StructLevel) {
	p := sl.Current().Interface().(Plan)
	seen := make(map[string]int, len(p.Steps))
	for i, step := range p.Steps {
		if step.ID == "" {
			continue
		}
		if first, ok := seen[step.ID]; ok {
			sl.ReportError(
				step.ID,
				fmt.Sprintf("steps[%d].id", i),
				fmt.Sprintf("Steps[%d].ID", i),
				"duplicate",
				fmt.Sprintf("steps[%d].id", first),
			)
			continue
		}
		seen[step.ID] = i
	}
}

func validateReview(sl validator.StructLevel) {
	r := sl.Current().Interface().(Review)
	switch r.Decision {
	case ReviewReject:
		if len(r.Issues) == 0 {
			sl.ReportError(r.Issues, "issues", "Issues", "issues_required", "")
		}
	case ReviewAccept:
		if len(r.Issues) != 0 {
			sl.ReportError(r.Issues, "issues", "Issues", "issues_forbidden", "")
		}
	}
}
