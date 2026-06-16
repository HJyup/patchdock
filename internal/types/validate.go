package types

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
)

var contractValidator = newContractValidator()

func newContractValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonFieldName)
	v.RegisterStructValidation(validatePlan, Plan{})
	v.RegisterStructValidation(validateReview, Review{})
	return v
}

func jsonFieldName(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("json"), ",")[0]
	switch name {
	case "", "-":
		return f.Name
	default:
		return name
	}
}

func validateStruct(value any, root string) error {
	if value == nil {
		return fmt.Errorf("%s: nil", root)
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return fmt.Errorf("%s: nil", root)
	}

	err := contractValidator.Struct(value)
	if err == nil {
		return nil
	}

	if invalid, ok := errors.AsType[*validator.InvalidValidationError](err); ok {
		return fmt.Errorf("%s: %w", root, invalid)
	}

	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) {
		return err
	}

	errs := make([]error, 0, len(fieldErrors))
	for _, fieldErr := range fieldErrors {
		errs = append(errs, translate(fieldErr, root))
	}
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Error() < errs[j].Error()
	})
	return errors.Join(errs...)
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

func translate(fieldErr validator.FieldError, root string) error {
	path := errorPath(fieldErr.Namespace(), root)
	switch fieldErr.Tag() {
	case "required", "min":
		return fmt.Errorf("%s: empty", path)
	case "gte":
		return fmt.Errorf("%s: must be >= %s", path, fieldErr.Param())
	case "oneof":
		return fmt.Errorf("%s: invalid value %q", path, fieldErr.Value())
	case "duplicate":
		return fmt.Errorf("%s: duplicate of %s", path, fieldErr.Param())
	case "patch_required":
		return fmt.Errorf("%s: empty for %s status", path, fieldErr.Param())
	case "issues_required":
		return fmt.Errorf("%s: required when decision is reject", path)
	case "issues_forbidden":
		return fmt.Errorf("%s: must be empty when decision is accept", path)
	default:
		if fieldErr.Param() == "" {
			return fmt.Errorf("%s: failed %s validation", path, fieldErr.Tag())
		}
		return fmt.Errorf("%s: failed %s=%s validation", path, fieldErr.Tag(), fieldErr.Param())
	}
}

func errorPath(namespace, root string) string {
	if namespace == "" {
		return root
	}
	if dot := strings.Index(namespace, "."); dot >= 0 {
		return root + namespace[dot:]
	}
	return root
}
