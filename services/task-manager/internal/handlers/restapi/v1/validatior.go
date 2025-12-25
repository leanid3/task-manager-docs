package v1

import (
	"app/pkg/response"

	"github.com/go-playground/validator/v10"
)

func parseValidationErrors(err error) []response.ValidationError {
	var validationErrors []response.ValidationError
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			validationErrors = append(validationErrors, response.ValidationError{
				Field:   fe.Field(),
				Message: getValidationMessage(fe),
				Tag:     fe.Tag(),
			})
		}
	}
	return validationErrors
}

// getValidationMessage возвращает человекочитаемое сообщение об ошибке
func getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " обязательно для заполнения"
	case "min":
		return fe.Field() + " должно быть не менее " + fe.Param()
	case "max":
		return fe.Field() + " должно быть не более " + fe.Param()
	case "oneof":
		return fe.Field() + " должно быть одним из: " + fe.Param()
	case "email":
		return fe.Field() + " должно быть действительным email"
	}
	return fe.Field() + " некорректное значение"
}
