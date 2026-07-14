package utils

import "github.com/go-playground/validator/v10"

// Validate is a global validator instance
var Validate = validator.New()

// ErrorResponse represents custom validation error payload
type ErrorResponse struct {
	FailedField string `json:"failed_field"`
	Tag         string `json:"tag"`
	Value       string `json:"value"`
}

// ValidateStruct validates a struct and returns custom formatted errors
func ValidateStruct(s interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	err := Validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}
	return errors
}
