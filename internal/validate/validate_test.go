package validate_test

import (
	"fmt"
	"testing"

	"github.com/HJyup/patchdock/internal/validate"
	"github.com/go-playground/validator/v10"
)

type jsonSubject struct {
	Name  string `json:"name" validate:"required"`
	Count int    `json:"count" validate:"gte=0"`
	Kind  string `json:"kind" validate:"omitempty,oneof=alpha beta"`
	Code  string `json:"code" validate:"omitempty,len=3"`
}

type yamlSubject struct {
	Name string `yaml:"name" validate:"required"`
}

func TestStructPassesValidValue(t *testing.T) {
	v := validate.New("json", nil)
	if err := v.Struct(jsonSubject{Name: "ok"}, "subject"); err != nil {
		t.Fatalf("valid value rejected: %v", err)
	}
}

func TestDefaultMessagesAndSortedJoin(t *testing.T) {
	v := validate.New("json", nil)

	err := v.Struct(jsonSubject{Count: -1, Kind: "gamma"}, "subject")
	want := "subject.count: must be >= 0\n" +
		"subject.kind: invalid value \"gamma\"\n" +
		"subject.name: empty"
	assertError(t, err, want)
}

func TestUnknownTagFallsBackToGenericMessage(t *testing.T) {
	v := validate.New("json", nil)

	err := v.Struct(jsonSubject{Name: "ok", Code: "toolong"}, "subject")
	assertError(t, err, "subject.code: failed len=3 validation")
}

func TestTranslatorOverridesDefault(t *testing.T) {
	v := validate.New("json", map[string]validate.Translator{
		"required": func(path string, _ validator.FieldError) error {
			return fmt.Errorf("%s: give me a value", path)
		},
	})

	err := v.Struct(jsonSubject{}, "subject")
	assertError(t, err, "subject.name: give me a value")
}

func TestTagKeySelectsFieldNames(t *testing.T) {
	v := validate.New("yaml", nil)

	err := v.Struct(yamlSubject{}, "cfg")
	assertError(t, err, "cfg.name: empty")
}

func TestMissingTagFallsBackToGoFieldName(t *testing.T) {
	// A json-tagged struct validated with yaml naming has no yaml tags,
	// so paths fall back to the Go field name.
	v := validate.New("yaml", nil)

	err := v.Struct(jsonSubject{Count: -1}, "subject")
	want := "subject.Count: must be >= 0\n" +
		"subject.Name: empty"
	assertError(t, err, want)
}

func TestNilGuards(t *testing.T) {
	v := validate.New("json", nil)

	assertError(t, v.Struct(nil, "subject"), "subject: nil")

	var s *jsonSubject
	assertError(t, v.Struct(s, "subject"), "subject: nil")
}

func assertError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if err.Error() != want {
		t.Fatalf("error mismatch\n got: %q\nwant: %q", err.Error(), want)
	}
}
