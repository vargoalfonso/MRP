// Package validator wraps go-playground/validator with structured, frontend-friendly error output.
package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldError represents a single field validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

var instance = validator.New()

func init() {
	// Use JSON tag name as field key so response matches request body keys.
	instance.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Validate runs struct validation and returns a slice of FieldError on failure,
// or nil when the struct is valid.
func Validate(s interface{}) []FieldError {
	err := instance.Struct(s)
	if err == nil {
		return nil
	}

	var errs validator.ValidationErrors
	if !isValidationErrors(err, &errs) {
		return []FieldError{{Field: "_", Message: err.Error()}}
	}

	out := make([]FieldError, 0, len(errs))
	for _, fe := range errs {
		out = append(out, FieldError{
			Field:   fe.Field(),
			Message: humanMessage(fe),
		})
	}
	return out
}

// humanMessage converts a validator.FieldError into a readable sentence.
func humanMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fe.Field())
	case "min":
		if fe.Type().Kind().String() == "string" {
			return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
		}
		return fmt.Sprintf("%s must be at least %s", fe.Field(), fe.Param())
	case "max":
		if fe.Type().Kind().String() == "string" {
			return fmt.Sprintf("%s must be at most %s characters", fe.Field(), fe.Param())
		}
		return fmt.Sprintf("%s must be at most %s", fe.Field(), fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), strings.ReplaceAll(fe.Param(), " ", ", "))
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fe.Field())
	case "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	case "numeric":
		return fmt.Sprintf("%s must contain only numbers", fe.Field())
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", fe.Field())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Field(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fe.Field(), fe.Param())
	case "lt":
		return fmt.Sprintf("%s must be less than %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}

func isValidationErrors(err error, target *validator.ValidationErrors) bool {
	v, ok := err.(validator.ValidationErrors)
	if ok {
		*target = v
	}
	return ok
}
