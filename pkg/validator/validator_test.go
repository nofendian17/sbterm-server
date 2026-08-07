package validator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type user struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
	Role  string `json:"role" validate:"oneof=admin user"`
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		in         any
		wantValid  bool
		wantFields map[string]string
	}{
		{
			name:      "valid struct",
			in:        user{Name: "Alice", Email: "alice@example.com", Age: 30, Role: "admin"},
			wantValid: true,
		},
		{
			name:       "missing required field reports json name",
			in:         user{Email: "alice@example.com", Age: 30, Role: "user"},
			wantFields: map[string]string{"name": "is required"},
		},
		{
			name:       "invalid email",
			in:         user{Name: "Alice", Email: "not-an-email", Age: 30, Role: "user"},
			wantFields: map[string]string{"email": "must be a valid email address"},
		},
		{
			name:       "age over upper bound",
			in:         user{Name: "Alice", Email: "alice@example.com", Age: 200, Role: "user"},
			wantFields: map[string]string{"age": "must be less than or equal to 130"},
		},
		{
			name:       "role not in oneof",
			in:         user{Name: "Alice", Email: "alice@example.com", Age: 30, Role: "root"},
			wantFields: map[string]string{"role": "must be one of: admin user"},
		},
		{
			name: "multiple failures reported together",
			in:   user{Email: "bad", Age: 30, Role: "root"},
			wantFields: map[string]string{
				"name":  "is required",
				"email": "must be a valid email address",
				"role":  "must be one of: admin user",
			},
		},
		{
			name:       "nil input",
			in:         nil,
			wantFields: map[string]string{"_": "input is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			err := v.Validate(tt.in)

			if tt.wantValid {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			ve, ok := AsValidationError(err)
			require.True(t, ok, "expected *ValidationError")
			assert.Equal(t, tt.wantFields, ve.Fields)
		})
	}
}

func TestValidateNonStruct(t *testing.T) {
	v := New()
	err := v.Validate("not a struct")

	require.Error(t, err)
	_, ok := AsValidationError(err)
	assert.False(t, ok, "non-struct input should not produce *ValidationError")
}

func TestValidationErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *ValidationError
		want string
	}{
		{
			name: "empty fields",
			err:  &ValidationError{Fields: map[string]string{}},
			want: "validation failed",
		},
		{
			name: "fields sorted",
			err: &ValidationError{Fields: map[string]string{
				"zebra": "bad",
				"alpha": "bad",
			}},
			want: "alpha: bad; zebra: bad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestAsValidationError(t *testing.T) {
	ve := &ValidationError{Fields: map[string]string{"a": "b"}}
	var target *ValidationError
	require.True(t, errors.As(ve, &target))
	assert.Equal(t, ve, target)

	assert.False(t, func() bool {
		var out *ValidationError
		return errors.As(errors.New("plain"), &out)
	}())
}

func TestValidateWithOptions(t *testing.T) {
	type payload struct {
		Name string `form:"display_name" validate:"required"`
		Meta struct {
			ID string `form:"id" validate:"required"`
		} `form:"meta" validate:"required"`
	}

	t.Run("custom tag name", func(t *testing.T) {
		v := New(WithTagName("form"))
		err := v.Validate(payload{})

		ve, ok := AsValidationError(err)
		require.True(t, ok)
		assert.Equal(t, "is required", ve.Fields["display_name"])
	})

	t.Run("required struct can be disabled", func(t *testing.T) {
		type input struct {
			Meta struct{} `json:"meta" validate:"required"`
		}

		v := New(WithRequiredStructEnabled(false))
		err := v.Validate(input{})

		assert.NoError(t, err)
	})
}

func TestValidateMessageTags(t *testing.T) {
	type payload struct {
		Short string `json:"short" validate:"min=3"`
		Long  string `json:"long" validate:"max=3"`
		Code  string `json:"code" validate:"len=4"`
		URL   string `json:"url" validate:"url"`
		ID    string `json:"id" validate:"uuid"`
		Even  int    `json:"even" validate:"oneof=2 4 6"`
	}

	v := New()
	err := v.Validate(payload{
		Short: "ab",
		Long:  "abcd",
		Code:  "abc",
		URL:   "not-url",
		ID:    "not-uuid",
		Even:  3,
	})

	ve, ok := AsValidationError(err)
	require.True(t, ok)
	assert.Equal(t, map[string]string{
		"short": "must be at least 3",
		"long":  "must be at most 3",
		"code":  "must be exactly 4 characters",
		"url":   "must be a valid URL",
		"id":    "must be a valid UUID",
		"even":  "must be one of: 2 4 6",
	}, ve.Fields)
}

func TestValidateDefaultMessage(t *testing.T) {
	type payload struct {
		Value string `json:"value" validate:"startswith=APP_"`
	}

	v := New()
	err := v.Validate(payload{Value: "NOPE"})

	ve, ok := AsValidationError(err)
	require.True(t, ok)
	assert.Equal(t, map[string]string{"value": "failed on validation tag \"startswith\""}, ve.Fields)
}
