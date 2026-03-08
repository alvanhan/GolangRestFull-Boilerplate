package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

type Validator struct {
	v *validator.Validate
}

// New returns a Validator with custom validators registered.
func New() *Validator {
	v := validator.New()

	// username: alphanumeric + underscore only
	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return usernameRegex.MatchString(fl.Field().String())
	})

	// strong_password: min 8 chars, has upper, lower, digit, special char
	_ = v.RegisterValidation("strong_password", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 {
			return false
		}
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		for _, ch := range password {
			switch {
			case unicode.IsUpper(ch):
				hasUpper = true
			case unicode.IsLower(ch):
				hasLower = true
			case unicode.IsDigit(ch):
				hasDigit = true
			case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
				hasSpecial = true
			}
		}
		return hasUpper && hasLower && hasDigit && hasSpecial
	})

	return &Validator{v: v}
}

// Validate runs the validator on i and returns the first error as a string,
// or nil if the struct is valid.
func (v *Validator) Validate(i interface{}) error {
	return v.v.Struct(i)
}

// ValidateStruct validates i and returns a map of field name → human-readable
// error message for every violated constraint.
func (v *Validator) ValidateStruct(s interface{}) map[string]string {
	errs := v.v.Struct(s)
	if errs == nil {
		return nil
	}

	validationErrors, ok := errs.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"error": errs.Error()}
	}

	result := make(map[string]string, len(validationErrors))
	for _, fe := range validationErrors {
		field := strings.ToLower(fe.Field())
		result[field] = fieldErrorMessage(fe)
	}
	return result
}

func fieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "username":
		return "must contain only letters, digits, and underscores"
	case "strong_password":
		return "must be at least 8 characters and contain uppercase, lowercase, digit, and special character"
	case "url":
		return "must be a valid URL"
	default:
		return fmt.Sprintf("failed validation: %s", fe.Tag())
	}
}
