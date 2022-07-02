package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type Validate = validator.Validate
type ValidationErrors = validator.ValidationErrors

func NewValidator(locale string) (*Validate, ut.Translator, error) {
	translator := en.New()
	uni := ut.New(translator, translator)

	trans, found := uni.GetTranslator(locale)
	if !found {
		return nil, nil, fmt.Errorf("%s translator not found", locale)
	}

	v := validator.New()

	if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
		return nil, nil, err
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return v, trans, nil
}

func ExtractErrorMap(trans ut.Translator, err error) map[string]string {
	res := make(map[string]string)

	for _, e := range err.(validator.ValidationErrors) {
		res[e.Field()] = e.Translate(trans)
	}

	return res
}
