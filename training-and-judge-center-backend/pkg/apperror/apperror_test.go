package apperror_test

import (
	"encoding/json"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestFieldError_CodeOmittedWhenEmpty(t *testing.T) {
	b, err := json.Marshal(apperror.FieldError{Field: "file", Message: "bad"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"field":"file","message":"bad"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFieldError_CodeSerializedWhenSet(t *testing.T) {
	b, err := json.Marshal(apperror.FieldError{Field: "file", Code: "ZIP_TOO_LARGE", Message: "bad"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"field":"file","code":"ZIP_TOO_LARGE","message":"bad"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
