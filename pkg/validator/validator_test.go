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
	tests := []struct {
		name string
		in   any
	}{
		{name: "string input is rejected", in: "not a struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			err := v.Validate(tt.in)

			require.Error(t, err)
			_, ok := AsValidationError(err)
			assert.False(t, ok, "non-struct input should not produce *ValidationError")
		})
	}
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
	tests := []struct {
		name string
		err  error
		want *ValidationError
	}{
		{
			name: "matches validation error",
			err:  &ValidationError{Fields: map[string]string{"a": "b"}},
			want: &ValidationError{Fields: map[string]string{"a": "b"}},
		},
		{
			name: "does not match plain error",
			err:  errors.New("plain"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target *ValidationError
			if tt.want != nil {
				require.True(t, errors.As(tt.err, &target))
				assert.Equal(t, tt.want, target)
				return
			}
			assert.False(t, errors.As(tt.err, &target))
		})
	}
}

func TestValidateWithOptions(t *testing.T) {
	type payload struct {
		Name string `form:"display_name" validate:"required"`
		Meta struct {
			ID string `form:"id" validate:"required"`
		} `form:"meta" validate:"required"`
	}
	type disabled struct {
		Meta struct{} `json:"meta" validate:"required"`
	}

	tests := []struct {
		name       string
		opts       []Option
		in         any
		wantFields map[string]string // nil means expect success
	}{
		{
			name:       "custom tag name",
			opts:       []Option{WithTagName("form")},
			in:         payload{},
			wantFields: map[string]string{"display_name": "is required"},
		},
		{
			name: "required struct can be disabled",
			opts: []Option{WithRequiredStructEnabled(false)},
			in:   disabled{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New(tt.opts...)
			err := v.Validate(tt.in)

			if tt.wantFields == nil {
				assert.NoError(t, err)
				return
			}

			ve, ok := AsValidationError(err)
			require.True(t, ok)
			for field, want := range tt.wantFields {
				assert.Equal(t, want, ve.Fields[field])
			}
		})
	}
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

	tests := []struct {
		name string
		in   any
		want map[string]string
	}{
		{
			name: "built-in tags produce friendly messages",
			in: payload{
				Short: "ab",
				Long:  "abcd",
				Code:  "abc",
				URL:   "not-url",
				ID:    "not-uuid",
				Even:  3,
			},
			want: map[string]string{
				"short": "must be at least 3",
				"long":  "must be at most 3",
				"code":  "must be exactly 4 characters",
				"url":   "must be a valid URL",
				"id":    "must be a valid UUID",
				"even":  "must be one of: 2 4 6",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			err := v.Validate(tt.in)

			ve, ok := AsValidationError(err)
			require.True(t, ok)
			assert.Equal(t, tt.want, ve.Fields)
		})
	}
}

func TestValidateDefaultMessage(t *testing.T) {
	type payload struct {
		Value string `json:"value" validate:"startswith=APP_"`
	}

	tests := []struct {
		name string
		in   any
		want map[string]string
	}{
		{
			name: "unknown tag yields default message",
			in:   payload{Value: "NOPE"},
			want: map[string]string{"value": `failed on validation tag "startswith"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			err := v.Validate(tt.in)

			ve, ok := AsValidationError(err)
			require.True(t, ok)
			assert.Equal(t, tt.want, ve.Fields)
		})
	}
}
