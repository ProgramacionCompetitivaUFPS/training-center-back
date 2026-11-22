package problem

import (
	"errors"
	"reflect"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestValidateIfSet_SkipsAFieldThatWasNotSent(t *testing.T) {
	var fieldErrs []apperror.FieldError

	got, err := validateIfSet(&fieldErrs, (*string)(nil), func(string) (string, error) {
		t.Fatal("the builder must not run when the field was not sent")
		return "", nil
	})

	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", got, err)
	}
	if len(fieldErrs) != 0 {
		t.Errorf("expected no field errors, got %v", fieldErrs)
	}
}

func TestValidateIfSet_ReturnsTheBuiltValue(t *testing.T) {
	var fieldErrs []apperror.FieldError
	in := "abc"

	got, err := validateIfSet(&fieldErrs, &in, func(v string) (int, error) { return len(v), nil })

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || *got != 3 {
		t.Fatalf("expected a pointer to 3, got %v", got)
	}
	if len(fieldErrs) != 0 {
		t.Errorf("expected no field errors, got %v", fieldErrs)
	}
}

func TestValidateIfSet_CollectsAValidationErrorAndReturnsNoValue(t *testing.T) {
	var fieldErrs []apperror.FieldError
	in := "abc"
	validation := apperror.NewValidation([]apperror.FieldError{{Field: "x", Message: "bad"}})

	got, err := validateIfSet(&fieldErrs, &in, func(string) (int, error) { return 0, validation })

	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", got, err)
	}
	if !reflect.DeepEqual(fieldErrs, []apperror.FieldError{{Field: "x", Message: "bad"}}) {
		t.Errorf("unexpected field errors: %v", fieldErrs)
	}
}

func TestValidateIfSet_AnUnexpectedErrorBecomesInternalAndIsNotCollected(t *testing.T) {
	var fieldErrs []apperror.FieldError
	in := "abc"

	got, err := validateIfSet(&fieldErrs, &in, func(string) (int, error) { return 0, errors.New("boom") })

	if got != nil {
		t.Errorf("expected no value, got %v", got)
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Fatalf("expected an internal AppError, got %v", err)
	}
	if len(fieldErrs) != 0 {
		t.Errorf("expected no field errors, got %v", fieldErrs)
	}
}
