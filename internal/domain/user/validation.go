package user

import (
	"fmt"
	"regexp"
	"unicode"
)

// ValidationError shows a single validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface for ValidationError
func (ve ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", ve.Field, ve.Message)
}

// ValidationErrors represents a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}

	msg := "validation errors: "
	for i, err := range ve {
		if i > 0 {
			msg += ", "
		}
		msg += err.Message
	}
	return msg
}

// HasErrors checks if there are any validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Constants for validation rules
const (
	MinLoginLength    = 4
	MinPasswordLength = 6
	MaxLoginLength    = 50
	MaxPasswordLength = 128
)

// ValidateLogin validates the login according to business rules
func ValidateLogin(login string) ValidationErrors {
	var errors ValidationErrors

	if login == "" {
		errors = append(errors, ValidationError{
			Field:   "login",
			Message: "login cannot be empty",
		})
		return errors
	}

	if len(login) < MinLoginLength {
		errors = append(errors, ValidationError{
			Field:   "login",
			Message: fmt.Sprintf("login must be at least %d characters long", MinLoginLength),
		})
	}

	if len(login) > MaxLoginLength {
		errors = append(errors, ValidationError{
			Field:   "login",
			Message: fmt.Sprintf("login cannot be longer than %d characters", MaxLoginLength),
		})
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(login) {
		errors = append(errors, ValidationError{
			Field:   "login",
			Message: "login can only contain letters, numbers, underscores and hyphens",
		})
	}

	return errors
}

// ValidatePassword validates the password according to business rules
func ValidatePassword(password string) ValidationErrors {
	var errors ValidationErrors

	if password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "password cannot be empty",
		})
		return errors
	}

	if len(password) < MinPasswordLength {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password must be at least %d characters long", MinPasswordLength),
		})
	}

	if len(password) > MaxPasswordLength {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password cannot be longer than %d characters", MaxPasswordLength),
		})
	}

	if !hasMinimumComplexityPassword(password) {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "password must contain at least one number and one letter",
		})
	}

	return errors
}

// hasMinimumComplexityPassword checks the minimum complexity of the password
func hasMinimumComplexityPassword(password string) bool {
	hasLetter := false
	hasNumber := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsNumber(char) {
			hasNumber = true
		}
		if hasLetter && hasNumber {
			return true
		}
	}

	return hasLetter && hasNumber
}

// ValidateUserCredentials validates the user credentials
func ValidateUserCredentials(login, password string) error {
	var allErrors ValidationErrors

	loginErrors := ValidateLogin(login)
	allErrors = append(allErrors, loginErrors...)

	passwordErrors := ValidatePassword(password)
	allErrors = append(allErrors, passwordErrors...)

	if allErrors.HasErrors() {
		return allErrors
	}

	return nil
}
