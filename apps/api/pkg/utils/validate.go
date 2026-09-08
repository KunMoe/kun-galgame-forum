package utils

import (
	"fmt"
	"net/url"
	"strings"

	"kun-galgame-api/pkg/errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	if err := v.RegisterValidation("downloadlink", isDownloadLink); err != nil {
		panic(err)
	}
	return v
}

// A download link is rendered straight into a KunLink `to`, so the scheme is an
// allowlist rather than "anything net/url accepts": javascript: and data: both
// parse, both carry a non-empty Opaque, and both passed the url tag this
// replaced.
var downloadLinkSchemes = map[string]bool{
	"http": true, "https": true,
	"ftp": true, "ftps": true,
	"magnet": true, "ed2k": true, "thunder": true,
}

// go-playground's url tag rejects a URI whose Host, Opaque and Fragment are all
// empty. A magnet URI is exactly that shape — everything lives in the query.
// Uploaders substituted a full-width ？, which makes the tail Opaque and
// validates, and produces a link no torrent client can use.
func isDownloadLink(fl validator.FieldLevel) bool {
	value := strings.TrimSpace(fl.Field().String())
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return false
	}
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if !downloadLinkSchemes[strings.ToLower(u.Scheme)] {
		return false
	}
	return u.Host != "" || u.Opaque != "" || u.RawQuery != "" || u.Fragment != ""
}

func ParseAndValidate(c fiber.Ctx, dst any) *errors.AppError {
	if err := c.Bind().Body(dst); err != nil {
		return errors.ErrBadRequest("请求格式错误")
	}
	if err := validate.Struct(dst); err != nil {
		return errors.ErrValidation(translateValidationErrors(err))
	}
	return nil
}

func ParseQueryAndValidate(c fiber.Ctx, dst any) *errors.AppError {
	if err := c.Bind().Query(dst); err != nil {
		return errors.ErrBadRequest("查询参数格式错误")
	}
	if err := validate.Struct(dst); err != nil {
		return errors.ErrValidation(translateValidationErrors(err))
	}
	return nil
}

func translateValidationErrors(err error) string {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return "请求参数验证失败"
	}

	messages := make([]string, 0, len(validationErrors))
	for _, fe := range validationErrors {
		messages = append(messages, translateFieldError(fe))
	}
	return strings.Join(messages, "; ")
}

func translateFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s 不能为空", field)
	case "email":
		return "邮箱格式不正确"
	case "min":
		return fmt.Sprintf("%s 长度不能小于 %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能大于 %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s 的值必须是 %s 之一", field, fe.Param())
	case "downloadlink":
		return fmt.Sprintf("%s 不是有效的下载链接", field)
	default:
		return fmt.Sprintf("%s 验证失败", field)
	}
}
