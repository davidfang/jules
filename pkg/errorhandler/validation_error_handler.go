// Package errorhandler 提供了处理特定类型错误的辅助函数。
package errorhandler

import (
	"go-base-system/pkg/response"   // 导入自定义的响应包
	"go-base-system/pkg/translator" // 导入自定义的翻译包

	"github.com/gin-gonic/gin"                // 导入 Gin 框架
	"github.com/go-playground/validator/v10" // 导入 validator v10
)

// HandleValidationErrors 函数用于处理 Gin 的参数绑定和校验错误。
// 如果错误是 validator.ValidationErrors 类型，它会翻译错误信息并使用统一的格式发送响应。
// c: Gin 的上下文对象。
// err: 需要处理的错误。
// 返回: 如果错误是 validator.ValidationErrors 并已处理，则返回 true；否则返回 false。
func HandleValidationErrors(c *gin.Context, err error) bool {
	// 1. 检查传入的错误是否为 nil
	if err == nil {
		return false // 如果没有错误，则无需处理
	}

	// 2. 使用类型断言检查错误是否为 validator.ValidationErrors
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 如果错误不是 validator.ValidationErrors 类型，则不由该函数处理
		return false
	}

	// 3. 如果是 validator.ValidationErrors，则翻译错误信息
	// 调用 translator.TranslateValidationErrors 获取翻译后的错误详情
	translatedErrors := translator.TranslateValidationErrors(validationErrs)

	// 4. 发送包含校验错误详情的响应
	// 使用 pkg/response 包中的 FailWithValidationErrors 函数发送标准化的错误响应
	// 业务状态码 4001 (示例) 代表输入参数校验失败
	// "输入参数校验失败" 是给前端或用户的主提示信息
	// translatedErrors 是包含各字段具体错误信息的 map
	response.FailWithValidationErrors(c, 4001, "输入参数校验失败", translatedErrors)

	// 5. 返回 true，表示错误已处理
	return true
}
