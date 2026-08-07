// Package validator wraps go-playground/validator into a small interface and
// normalizes validation failures into a field -> message map ready for the
// response envelope (pkg/response.ValidationError).
package validator

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	govalidator "github.com/go-playground/validator/v10"
)

type Validator interface {
	Validate(s any) error
}

type Option func(*options)

type options struct {
	tagName        string
	requiredStruct bool
}

// WithTagName sets the struct tag used to name fields in validation errors.
// Defaults to "json".
func WithTagName(tag string) Option {
	return func(o *options) { o.tagName = tag }
}

// WithRequiredStructEnabled toggles validator.WithRequiredStructEnabled.
// Defaults to true (the default behaviour in validator v11+).
func WithRequiredStructEnabled(enabled bool) Option {
	return func(o *options) { o.requiredStruct = enabled }
}

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(&b, "%s: %s", k, e.Fields[k])
	}
	return b.String()
}

func AsValidationError(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}

type impl struct {
	v *govalidator.Validate
}

func New(opts ...Option) Validator {
	o := &options{
		tagName:        "json",
		requiredStruct: true,
	}
	for _, opt := range opts {
		opt(o)
	}

	var vopts []govalidator.Option
	if o.requiredStruct {
		vopts = append(vopts, govalidator.WithRequiredStructEnabled())
	}
	v := govalidator.New(vopts...)

	if o.tagName != "" {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get(o.tagName), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}

	return &impl{v: v}
}

func (im *impl) Validate(s any) error {
	if s == nil {
		return &ValidationError{Fields: map[string]string{"_": "input is required"}}
	}
	if err := im.v.Struct(s); err != nil {
		var ves govalidator.ValidationErrors
		if errors.As(err, &ves) {
			fields := make(map[string]string, len(ves))
			for _, fe := range ves {
				fields[fe.Field()] = messageFor(fe)
			}
			return &ValidationError{Fields: fields}
		}
		return err
	}
	return nil
}

func messageFor(fe govalidator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("failed on validation tag %q", fe.Tag())
	}
}
