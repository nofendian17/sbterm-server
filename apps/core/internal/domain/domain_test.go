package domain

import (
	"errors"
	"reflect"
	"testing"
)

func TestSentinelErrorsAreDistinct(t *testing.T) {
	sentinels := []error{
		ErrUserNotFound,
		ErrEmailTaken,
		ErrInvalidCredentials,
		ErrExpired,
		ErrSuspended,
		ErrPermissionDenied,
		ErrDuplicateWatchlist,
		ErrRoleNotFound,
		ErrPermissionNotFound,
	}

	for i, err := range sentinels {
		if err == nil {
			t.Fatalf("sentinel at index %d is nil", i)
		}
		// Each sentinel must identify as itself.
		if !errors.Is(err, err) {
			t.Errorf("expected errors.Is(%v, %v) to be true", err, err)
		}
		// No sentinel should match any other sentinel.
		for j, other := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(err, other) {
				t.Errorf("sentinel %v unexpectedly matched %v", err, other)
			}
		}
	}
}

func TestStructsCompileWithValidateTags(t *testing.T) {
	registerType := reflect.TypeOf(RegisterInput{})
	emailField, ok := registerType.FieldByName("Email")
	if !ok {
		t.Fatal("RegisterInput missing Email field")
	}
	if got := emailField.Tag.Get("validate"); got != "required,email" {
		t.Errorf("RegisterInput.Email validate tag = %q, want %q", got, "required,email")
	}

	passwordField, _ := registerType.FieldByName("Password")
	if got := passwordField.Tag.Get("validate"); got != "required,min=8" {
		t.Errorf("RegisterInput.Password validate tag = %q, want %q", got, "required,min=8")
	}

	addWLType := reflect.TypeOf(AddWatchlistInput{})
	symbolField, ok := addWLType.FieldByName("Symbol")
	if !ok {
		t.Fatal("AddWatchlistInput missing Symbol field")
	}
	if got := symbolField.Tag.Get("validate"); got != "required" {
		t.Errorf("AddWatchlistInput.Symbol validate tag = %q, want %q", got, "required")
	}

	// Ensure the types are constructible without error.
	_ = LoginInput{Email: "a@b.com", Password: "secret"}
	_ = Watchlist{ID: "w1", UserID: "u1", Symbol: "BBCA"}
	_ = User{ID: "u1", Email: "a@b.com"}
	_ = Role{ID: "r1", Name: "admin"}
	_ = Permission{ID: "p1", Resource: "user", Action: "read", Name: "user:read"}
}
