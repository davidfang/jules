// Package errorhandler_test 包含了对 errorhandler 包的单元测试。
package errorhandler_test

import (
	"encoding/json"
	"errors" // 导入标准 errors 包
	"fmt"
	"go-base-system/pkg/errorhandler" // 被测试的包
	"go-base-system/pkg/response"     // 自定义响应包，用于验证响应结构
	"go-base-system/pkg/translator"   // 翻译包，用于初始化翻译器
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"                // Gin 框架
	"github.com/go-playground/validator/v10" // Validator 包
	"github.com/stretchr/testify/assert"    // Testify断言库
)

// TestMain 用于执行测试前的设置，如此处初始化翻译器。
// 这样可以确保所有测试用例运行时，翻译器都已准备就绪。
func TestMain(m *testing.M) {
	fmt.Println("Initializing translator for errorhandler tests...")
	// 初始化翻译器，这对于 HandleValidationErrors 测试中正确翻译错误消息至关重要。
	if err := translator.InitTranslator(); err != nil {
		panic(fmt.Sprintf("Test setup failed: Failed to initialize translator: %v", err))
	}
	fmt.Println("Translator initialized successfully for errorhandler tests.")
	// 运行测试
	m.Run()
}

// TestHandleValidationErrors 测试 HandleValidationErrors 函数的各种场景。
func TestHandleValidationErrors(t *testing.T) {
	// 设置 Gin 为测试模式，避免不必要的日志输出
	gin.SetMode(gin.TestMode)

	// --- 测试用例 1: 传入 nil 错误 ---
	t.Run("nil_error", func(t *testing.T) {
		// 创建一个 ResponseRecorder 来捕获响应
		w := httptest.NewRecorder()
		// 创建一个测试用的 Gin Context
		c, _ := gin.CreateTestContext(w)

		// 调用 HandleValidationErrors 并传入 nil
		handled := errorhandler.HandleValidationErrors(c, nil)

		// 断言:
		// 1. HandleValidationErrors 应返回 false，表示没有处理错误。
		assert.False(t, handled, "HandleValidationErrors should return false for nil error")
		// 2. 响应状态码应未被设置 (默认为 0，或由 CreateTestContext 初始化时的默认值，通常不会是 200 OK)
		//    或者更准确地说，不应是 HandleValidationErrors 会设置的 http.StatusBadRequest。
		//    如果 ResponseRecorder 的 Code 是 0，表示没有写入状态码。
		assert.Equal(t, 0, w.Code, "Response status code should not be set for nil error")
		// 3. 响应体应为空。
		assert.Empty(t, w.Body.String(), "Response body should be empty for nil error")
	})

	// --- 测试用例 2: 传入普通 error (非 validator.ValidationErrors) ---
	t.Run("generic_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		genericErr := errors.New("这是一个普通的错误")

		handled := errorhandler.HandleValidationErrors(c, genericErr)

		// 断言:
		// 1. 返回 false。
		assert.False(t, handled, "HandleValidationErrors should return false for a generic error")
		// 2. 状态码未被设置。
		assert.Equal(t, 0, w.Code, "Response status code should not be set for a generic error")
		// 3. 响应体为空。
		assert.Empty(t, w.Body.String(), "Response body should be empty for a generic error")
	})

	// --- 测试用例 3: 传入 validator.ValidationErrors ---
	t.Run("validation_errors", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// 构造一个 validator.ValidationErrors 示例
		// 1. 定义一个带校验规则的结构体
		type TestRequest struct {
			Username string `json:"username" validate:"required,min=3"`
			Email    string `json:"email_field" validate:"required,email"`
		}
		// 2. 创建一个违反规则的实例
		req := TestRequest{
			Username: "u",             // 违反 min=3
			Email:    "not-an-email", // 违反 email
		}
		// 3. 获取 Gin 的 validator 引擎 (已在 TestMain 中通过 translator.InitTranslator() 配置好)
		validate, _ := binding.Validator.Engine().(*validator.Validate)
		validationErrs, _ := validate.Struct(req).(validator.ValidationErrors) // 进行校验并断言为 ValidationErrors

		// 调用 HandleValidationErrors
		handled := errorhandler.HandleValidationErrors(c, validationErrs)

		// 断言:
		// 1. 返回 true，表示错误已被处理。
		assert.True(t, handled, "HandleValidationErrors should return true for validation errors")
		// 2. HTTP 状态码应为 http.StatusBadRequest (400)。
		assert.Equal(t, http.StatusBadRequest, w.Code, "Response status code should be 400 Bad Request for validation errors")

		// 3. 解析响应体并验证内容。
		var respData response.ResponseData // 使用我们项目中定义的标准响应结构
		err := json.Unmarshal(w.Body.Bytes(), &respData)
		assert.NoError(t, err, "Failed to unmarshal response body")

		// 3a. 验证 Code 和 Msg
		assert.Equal(t, 4001, respData.Code, "Response business code should be 4001")
		assert.Equal(t, "输入参数校验失败", respData.Msg, "Response message should be '输入参数校验失败'")

		// 3b. 验证 Data 字段 (包含翻译后的错误信息)
		//    Data 字段应该是一个 map[string]string
		errorDetails, ok := respData.Data.(map[string]interface{}) // JSON 解析 map 为 map[string]interface{}
		assert.True(t, ok, "Response data should be a map")

		// 将 map[string]interface{} 转换为 map[string]string 以便比较
		expectedErrorDetails := map[string]string{
			"username":     "username 长度不能少于 3 个字符。", // 从 translator_test.go 借鉴预期翻译
			"email_field": "email_field 必须是有效的邮箱地址。",
		}
		
		actualErrorDetails := make(map[string]string)
		for k, v := range errorDetails {
			s, isString := v.(string)
			if !isString {
				t.Errorf("Value for key %s in error details is not a string: %T", k, v)
				continue
			}
			actualErrorDetails[k] = s
		}

		assert.Equal(t, expectedErrorDetails, actualErrorDetails, "Translated error details in response data do not match")
	})

	// --- 测试用例 4: 传入空的 validator.ValidationErrors ---
	// 这种情况理论上不应该发生，因为如果校验通过，err 会是 nil。
	// 但为了覆盖，我们手动创建一个空的 ValidationErrors。
	t.Run("empty_validation_errors", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		emptyValidationErrs := validator.ValidationErrors{} // 空的 ValidationErrors 切片

		handled := errorhandler.HandleValidationErrors(c, emptyValidationErrs)

		assert.True(t, handled, "HandleValidationErrors should return true for empty validation errors")
		assert.Equal(t, http.StatusBadRequest, w.Code, "Status code should be 400 for empty validation errors")

		var respData response.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &respData)
		assert.NoError(t, err)

		assert.Equal(t, 4001, respData.Code)
		assert.Equal(t, "输入参数校验失败", respData.Msg)
		
		// Data 应该是一个空的 map
		errorDetails, ok := respData.Data.(map[string]interface{})
		assert.True(t, ok, "Data should be a map")
		assert.Empty(t, errorDetails, "Data should be an empty map for empty validation errors")
	})
}
// Okay, I've created `pkg/errorhandler/validation_error_handler_test.go`.

// Here's a summary of what I've done:

// 1.  **`TestMain`**: Added `TestMain` to ensure `translator.InitTranslator()` is called once before any tests in this package run. This is crucial because `HandleValidationErrors` relies on the translator being initialized to correctly translate validation messages.
// 2.  **`TestHandleValidationErrors`**: This is the main test function, broken down into subtests for different scenarios:
//     *   **`nil_error`**: Tests that passing `nil` as an error results in `false` being returned, no HTTP response written (status code 0), and an empty body.
//     *   **`generic_error`**: Tests that passing a standard `error` (not `validator.ValidationErrors`) also results in `false`, no HTTP response written, and an empty body.
//     *   **`validation_errors`**: This is the core test case.
//         *   It constructs a sample struct (`TestRequest`) with validation tags.
//         *   It creates an instance of this struct that violates these rules.
//         *   It uses Gin's validator engine (which should have been configured by `translator.InitTranslator()`) to get `validator.ValidationErrors`.
//         *   It calls `errorhandler.HandleValidationErrors` with a Gin test context and these errors.
//         *   It asserts that `true` is returned.
//         *   It verifies that the HTTP status code in the `httptest.ResponseRecorder` is `http.StatusBadRequest`.
//         *   It unmarshals the JSON response body into `response.ResponseData`.
//         *   It checks if `respData.Code` is `4001` and `respData.Msg` is `"输入参数校验失败"`.
//         *   It validates that `respData.Data` is a map containing the correctly translated error messages for the specific validation failures (e.g., `{"username": "username 长度不能少于 3 个字符。", "email_field": "email_field 必须是有效的邮箱地址。"}`).
//     *   **`empty_validation_errors`**: Tests the scenario of passing an empty `validator.ValidationErrors` slice. It should still be handled as a validation error (return `true`, status 400, code 4001, msg "输入参数校验失败"), but the `data` field in the response should be an empty map.

// All assertions are done using `github.com/stretchr/testify/assert`. Chinese comments are included.

// Now, let's proceed to Part 3: Updating tests for `internal/handler/user_handler_test.go`.
// First, I need to read the content of `internal/handler/user_handler_test.go`.
