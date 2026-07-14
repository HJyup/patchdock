package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/HJyup/patchdock/internal/types"
	"github.com/HJyup/patchdock/internal/validate"
	"github.com/go-playground/validator/v10"
)

var requiredStages = []types.StageName{types.StagePlanner, types.StageExecutor, types.StageReviewer}

var configValidator = newConfigValidator()

func newConfigValidator() *validate.Validator {
	v := validate.New("yaml", map[string]validate.Translator{
		"gt": func(path string, fieldErr validator.FieldError) error {
			return fmt.Errorf("%s: must be > %s", path, fieldErr.Param())
		},
		"min": func(path string, fieldErr validator.FieldError) error {
			return fmt.Errorf("%s: must contain at least %s item", path, fieldErr.Param())
		},
		"tsfile": func(path string, _ validator.FieldError) error {
			return fmt.Errorf("%s: must be a .ts file", path)
		},
		"stage_missing": func(path string, _ validator.FieldError) error {
			return fmt.Errorf("%s: missing", path)
		},
	})

	v.RegisterValidation("tsfile", validateTSFile)
	v.RegisterStructValidation(validateStages, Config{})
	return v
}

func (c *Config) Validate() error {
	return configValidator.Struct(c, "config")
}

func validateStages(sl validator.StructLevel) {
	c := sl.Current().Interface().(Config)
	for _, stage := range requiredStages {
		if _, ok := c.Stages[stage]; !ok {
			sl.ReportError(c.Stages, "stages."+string(stage), "Stages", "stage_missing", "")
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
