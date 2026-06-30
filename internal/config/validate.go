package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
)

var requiredStages = []string{"planner", "executor", "reviewer"}

var configValidator = newConfigValidator()

func newConfigValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(yamlFieldName)
	v.RegisterValidation("tsfile", validateTSFile)
	v.RegisterStructValidation(validateStages, Config{})
	return v
}

func yamlFieldName(f reflect.StructField) string {
	name := strings.Split(f.Tag.Get("yaml"), ",")[0]
	switch name {
	case "", "-":
		return f.Name
	default:
		return name
	}
}

func (c *Config) Validate() error {
	return validateStruct(c, "config")
}

func (c *Config) validate() error {
	if c == nil {
		return validateStruct(c, "config")
	}
	return c.Validate()
}

func validateStruct(value any, root string) error {
	if value == nil {
		return fmt.Errorf("%s: nil", root)
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return fmt.Errorf("%s: nil", root)
	}

	err := configValidator.Struct(value)
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

func validateStages(sl validator.StructLevel) {
	c := sl.Current().Interface().(Config)
	for _, stage := range requiredStages {
		if _, ok := c.Stages[stage]; !ok {
			sl.ReportError(c.Stages, "stages."+stage, "Stages", "stage_missing", "")
		}
	}
}

func validateTSFile(fl validator.FieldLevel) bool {
	path, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	return strings.EqualFold(filepath.Ext(path), ".ts")
}

func translate(fieldErr validator.FieldError, root string) error {
	path := errorPath(fieldErr.Namespace(), root)
	switch fieldErr.Tag() {
	case "required":
		return fmt.Errorf("%s: empty", path)
	case "gt":
		return fmt.Errorf("%s: must be > %s", path, fieldErr.Param())
	case "gte":
		return fmt.Errorf("%s: must be >= %s", path, fieldErr.Param())
	case "min":
		return fmt.Errorf("%s: must contain at least %s item", path, fieldErr.Param())
	case "oneof":
		return fmt.Errorf("%s: invalid value %q", path, fieldErr.Value())
	case "tsfile":
		return fmt.Errorf("%s: must be a .ts file", path)
	case "stage_missing":
		return fmt.Errorf("%s: missing", path)
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
