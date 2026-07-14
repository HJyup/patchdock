package validate

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Translator renders one failed validation into a human-readable error
type Translator func(path string, fieldErr validator.FieldError) error

type Validator struct {
	validate    *validator.Validate
	translators map[string]Translator
}

func New(tagKey string, translators map[string]Translator) *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get(tagKey), ",")
		switch name {
		case "", "-":
			return f.Name
		default:
			return name
		}
	})

	return &Validator{validate: v, translators: translators}
}

// RegisterValidation adds a custom field-level validation for tag.
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// RegisterStructValidation adds a cross-field validation for the given types.
func (v *Validator) RegisterStructValidation(fn validator.StructLevelFunc, types ...any) {
	v.validate.RegisterStructValidation(fn, types...)
}

// Struct validates value and translates every field failure. The result is
// nil, or all failures joined in sorted order so output is deterministic.
func (v *Validator) Struct(value any, root string) error {
	if value == nil {
		return fmt.Errorf("%s: nil", root)
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return fmt.Errorf("%s: nil", root)
	}

	err := v.validate.Struct(value)
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
		errs = append(errs, v.translate(fieldErr, root))
	}
	slices.SortFunc(errs, func(a, b error) int {
		return strings.Compare(a.Error(), b.Error())
	})
	return errors.Join(errs...)
}

func (v *Validator) translate(fieldErr validator.FieldError, root string) error {
	path := errorPath(fieldErr.Namespace(), root)

	if t, ok := v.translators[fieldErr.Tag()]; ok {
		return t(path, fieldErr)
	}

	switch fieldErr.Tag() {
	case "required":
		return fmt.Errorf("%s: empty", path)
	case "gte":
		return fmt.Errorf("%s: must be >= %s", path, fieldErr.Param())
	case "oneof":
		return fmt.Errorf("%s: invalid value %q", path, fieldErr.Value())
	default:
		if fieldErr.Param() == "" {
			return fmt.Errorf("%s: failed %s validation", path, fieldErr.Tag())
		}
		return fmt.Errorf("%s: failed %s=%s validation", path, fieldErr.Tag(), fieldErr.Param())
	}
}

// errorPath swaps the struct-type prefix of a validator namespace
// ("Config.stages[planner]") for the caller's root label
// ("config.stages[planner]").
func errorPath(namespace, root string) string {
	if namespace == "" {
		return root
	}
	if dot := strings.Index(namespace, "."); dot >= 0 {
		return root + namespace[dot:]
	}
	return root
}
