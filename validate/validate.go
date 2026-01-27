// Package validate provides struct validation for go-micro.
// Similar to Bean Validation (Java) or ActiveModel Validations (Rails).
package validate

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
	Tag     string
	Value   interface{}
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator validates structs based on tags.
type Validator struct {
	tagName    string
	validators map[string]ValidatorFunc
}

// ValidatorFunc is a custom validation function.
type ValidatorFunc func(field reflect.Value, param string) bool

// New creates a new validator.
func New() *Validator {
	v := &Validator{
		tagName:    "validate",
		validators: make(map[string]ValidatorFunc),
	}
	// Register built-in validators
	v.RegisterValidator("required", validateRequired)
	v.RegisterValidator("email", validateEmail)
	v.RegisterValidator("min", validateMin)
	v.RegisterValidator("max", validateMax)
	v.RegisterValidator("len", validateLen)
	v.RegisterValidator("url", validateURL)
	v.RegisterValidator("alpha", validateAlpha)
	v.RegisterValidator("alphanum", validateAlphaNum)
	v.RegisterValidator("numeric", validateNumeric)
	v.RegisterValidator("uuid", validateUUID)
	return v
}

// RegisterValidator registers a custom validator.
func (v *Validator) RegisterValidator(name string, fn ValidatorFunc) {
	v.validators[name] = fn
}

// Validate validates a struct.
func (v *Validator) Validate(s interface{}) ValidationErrors {
	var errors ValidationErrors

	val := reflect.ValueOf(s)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return errors
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		tag := fieldType.Tag.Get(v.tagName)
		if tag == "" || tag == "-" {
			continue
		}

		fieldName := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}

			parts := strings.SplitN(rule, "=", 2)
			ruleName := parts[0]
			param := ""
			if len(parts) > 1 {
				param = parts[1]
			}

			validatorFn, ok := v.validators[ruleName]
			if !ok {
				continue
			}

			if !validatorFn(field, param) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: formatMessage(ruleName, param),
					Tag:     ruleName,
					Value:   field.Interface(),
				})
			}
		}
	}

	return errors
}

// Default validator instance.
var defaultValidator = New()

// Validate validates a struct using the default validator.
func Validate(s interface{}) ValidationErrors {
	return defaultValidator.Validate(s)
}

// RegisterValidator registers a custom validator on the default instance.
func RegisterValidator(name string, fn ValidatorFunc) {
	defaultValidator.RegisterValidator(name, fn)
}

// Built-in validators

func validateRequired(field reflect.Value, _ string) bool {
	switch field.Kind() {
	case reflect.String:
		return strings.TrimSpace(field.String()) != ""
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan:
		return !field.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return field.Float() != 0
	case reflect.Bool:
		return field.Bool()
	default:
		return true
	}
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func validateEmail(field reflect.Value, _ string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	return emailRegex.MatchString(field.String())
}

func validateMin(field reflect.Value, param string) bool {
	min := parseInt(param)
	switch field.Kind() {
	case reflect.String:
		return utf8.RuneCountInString(field.String()) >= min
	case reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() >= min
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() >= int64(min)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() >= uint64(min)
	case reflect.Float32, reflect.Float64:
		return field.Float() >= float64(min)
	default:
		return true
	}
}

func validateMax(field reflect.Value, param string) bool {
	max := parseInt(param)
	switch field.Kind() {
	case reflect.String:
		return utf8.RuneCountInString(field.String()) <= max
	case reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() <= max
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() <= int64(max)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() <= uint64(max)
	case reflect.Float32, reflect.Float64:
		return field.Float() <= float64(max)
	default:
		return true
	}
}

func validateLen(field reflect.Value, param string) bool {
	expected := parseInt(param)
	switch field.Kind() {
	case reflect.String:
		return utf8.RuneCountInString(field.String()) == expected
	case reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() == expected
	default:
		return true
	}
}

var urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

func validateURL(field reflect.Value, _ string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	return urlRegex.MatchString(field.String())
}

var alphaRegex = regexp.MustCompile(`^[a-zA-Z]+$`)

func validateAlpha(field reflect.Value, _ string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	return alphaRegex.MatchString(field.String())
}

var alphaNumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func validateAlphaNum(field reflect.Value, _ string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	return alphaNumRegex.MatchString(field.String())
}

var numericRegex = regexp.MustCompile(`^[0-9]+$`)

func validateNumeric(field reflect.Value, _ string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	return numericRegex.MatchString(field.String())
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validateUUID(field reflect.Value, _ string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	return uuidRegex.MatchString(field.String())
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func formatMessage(rule, param string) string {
	switch rule {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must be at least %s", param)
	case "max":
		return fmt.Sprintf("must be at most %s", param)
	case "len":
		return fmt.Sprintf("must be exactly %s", param)
	case "url":
		return "must be a valid URL"
	case "alpha":
		return "must contain only letters"
	case "alphanum":
		return "must contain only letters and numbers"
	case "numeric":
		return "must contain only numbers"
	case "uuid":
		return "must be a valid UUID"
	default:
		return "is invalid"
	}
}
