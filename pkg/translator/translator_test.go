// Package translator_test 包含了对 translator 包的单元测试。
package translator_test

import (
	"fmt"
	"go-base-system/pkg/translator" // 被测试的包
	"strings"
	"testing"

	"github.com/go-playground/validator/v10" // 导入 validator 包
	"github.com/stretchr/testify/assert"    // 导入 testify 断言库
)

// TestDTO 是一个用于测试多种校验规则的结构体。
// 字段使用了 json 标签，以便测试 RegisterTagNameFunc 的效果。
type TestDTO struct {
	Username        string   `json:"username" validate:"required,alphanum,min=3,max=20"`
	Password        string   `json:"password" validate:"required,min=6,max=30"`
	ConfirmPassword string   `json:"confirm_password" validate:"required,eqfield=Password"`
	Email           string   `json:"email_field" validate:"required,email"`
	Age             int      `json:"age" validate:"min=18,max=100"`
	Website         string   `json:"website_url" validate:"required,url"`
	Role            string   `json:"role" validate:"required,oneof=admin user guest"`
	Bio             string   `json:"biography" validate:"max=200"` // 测试只有 max
	Code            string   `json:"security_code" validate:"len=6"` // 测试固定长度
	Items           []string `json:"items_list" validate:"min=1,max=5"` // 测试切片长度
	RetryAttempts   int      `json:"retry_attempts" validate:"gte=0,lte=5"`
	OldField        string   `json:"old_field" validate:"nefield=Password"`
	Score           int      `json:"score" validate:"gt=0,lt=100"`
}

// init 函数在包级别执行，确保在所有测试运行之前初始化翻译器。
func init() {
	fmt.Println("Initializing translator for tests...")
	if err := translator.InitTranslator(); err != nil {
		panic(fmt.Sprintf("Failed to initialize translator for tests: %v", err))
	}
	fmt.Println("Translator initialized successfully for tests.")
}

// TestInitTranslator 测试 translator.InitTranslator 是否成功初始化了全局翻译器实例。
func TestInitTranslator(t *testing.T) {
	// arrange: init() 函数已经调用了 translator.InitTranslator()

	// act & assert
	assert.NotNil(t, translator.Trans, "全局翻译器 translator.Trans 不应为 nil")
	// 进一步检查 Trans 是否是期望的类型 (可选，因为 InitTranslator 内部逻辑复杂)
	// _, ok := translator.Trans.(ut.Translator)
	// assert.True(t, ok, "translator.Trans 应为 ut.Translator 类型")
}

// TestTranslateValidationErrors 测试 TranslateValidationErrors 函数的各种校验规则翻译。
func TestTranslateValidationErrors(t *testing.T) {
	validate := validator.New() // 创建一个新的校验器实例

	// 辅助函数，用于执行校验并获取错误
	getValidationErrors := func(dto interface{}) validator.ValidationErrors {
		err := validate.Struct(dto)
		if err == nil {
			return nil
		}
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			t.Fatalf("Expected validator.ValidationErrors, got %T", err)
		}
		return validationErrs
	}

	// 定义测试用例
	testCases := []struct {
		name          string      // 测试用例名称
		dtoInstance   interface{} // DTO 实例，包含待校验的数据
		expectedError string      // 预期的单个错误消息 (如果只校验一个字段)
		expectedMap   map[string]string // 预期的完整错误 map (如果校验多个字段或需要精确匹配)
	}{
		// --- required ---
		{
			name: "username_required",
			dtoInstance: &TestDTO{
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"username": "username 不能为空!"},
		},
		// --- alphanum ---
		{
			name: "username_alphanum",
			dtoInstance: &TestDTO{
				Username:        "user name!", // 包含空格和特殊字符
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"username": "username 只能包含字母和数字。"},
		},
		// --- min (string) ---
		{
			name: "username_min_length",
			dtoInstance: &TestDTO{
				Username:        "us", // 长度为 2, 要求 min=3
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"username": "username 长度不能少于 3 个字符。"},
		},
		// --- max (string) ---
		{
			name: "username_max_length",
			dtoInstance: &TestDTO{
				Username:        strings.Repeat("a", 21), // 长度为 21, 要求 max=20
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"username": "username 长度不能超过 20 个字符。"},
		},
		// --- email ---
		{
			name: "email_invalid",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "invalid-email", // 无效邮箱格式
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"email_field": "email_field 必须是有效的邮箱地址。"},
		},
		// --- eqfield ---
		{
			name: "confirm_password_eqfield",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "anotherPassword", // 与 Password 不一致
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"confirm_password": "confirm_password 必须等于 Password."},
		},
		// --- nefield ---
		{
			name: "old_field_nefield",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				OldField:        "pass123", // 与 Password 相同
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"old_field": "old_field 不能等于 Password."},
		},
		// --- min (number) ---
		{
			name: "age_min",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Age:             17, // 小于 18
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"age": "age 不能小于 18。"},
		},
		// --- max (number) ---
		{
			name: "age_max",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Age:             101, // 大于 100
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"age": "age 不能大于 100。"},
		},
		// --- url ---
		{
			name: "website_url_invalid",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "not a url", // 无效 URL
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"website_url": "website_url 必须是一个有效的URL。"},
		},
		// --- oneof ---
		{
			name: "role_oneof",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "superuser", // 不在 "admin user guest" 中
				Code:            "123456",
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"role": "role 必须是 [admin, user, guest] 中的一个。"},
		},
		// --- len (string) ---
		{
			name: "code_len_string",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "12345", // 长度为 5, 要求 len=6
				Items:           []string{"one"},
			},
			expectedMap: map[string]string{"security_code": "security_code 长度必须是 6 个字符。"},
		},
		// --- min (slice/items) ---
		{
			name: "items_min_items",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{}, // 0 项, 要求 min=1
			},
			expectedMap: map[string]string{"items_list": "items_list 不能少于 1 项。"},
		},
		// --- max (slice/items) ---
		{
			name: "items_max_items",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"1", "2", "3", "4", "5", "6"}, // 6 项, 要求 max=5
			},
			expectedMap: map[string]string{"items_list": "items_list 不能超过 5 项。"},
		},
		// --- gte (number) ---
		{
			name: "retry_attempts_gte",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
				RetryAttempts:   -1, // 小于 0
			},
			expectedMap: map[string]string{"retry_attempts": "retry_attempts 必须大于或等于 0."},
		},
		// --- lte (number) ---
		{
			name: "retry_attempts_lte",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
				RetryAttempts:   6, // 大于 5
			},
			expectedMap: map[string]string{"retry_attempts": "retry_attempts 必须小于或等于 5."},
		},
		// --- gt (number) ---
		{
			name: "score_gt",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
				Score:           0, // 等于 0, 要求大于 0
			},
			expectedMap: map[string]string{"score": "score 必须大于 0."},
		},
		// --- lt (number) ---
		{
			name: "score_lt",
			dtoInstance: &TestDTO{
				Username:        "validuser",
				Password:        "pass123",
				ConfirmPassword: "pass123",
				Email:           "test@example.com",
				Website:         "http://example.com",
				Role:            "admin",
				Code:            "123456",
				Items:           []string{"one"},
				Score:           100, // 等于 100, 要求小于 100
			},
			expectedMap: map[string]string{"score": "score 必须小于 100."},
		},
		// --- multiple errors ---
		{
			name: "multiple_errors",
			dtoInstance: &TestDTO{
				Username: "u!",     // alphanum, min
				Email:    "invalid", // email
				Age:      10,       // min
			},
			expectedMap: map[string]string{
				"username":    "username 只能包含字母和数字。", // alphanum 优先级可能高于 min，或取决于校验器内部顺序
				// 如果 alphanum 通过了，才会报 min。这里我们假设 alphanum 先报。
				// 如果希望测试多个错误，需要更精确的控制或接受 validator 的行为。
				// 通常 validator 对一个字段只报一个错误。
				// 为了测试多个字段的错误，我们让每个字段只违反一个规则。
				// 修正：让 username 仅违反 min
				// "username": "username 长度不能少于 3 个字符。",
				// "email_field": "email_field 必须是有效的邮箱地址。",
				// "age": "age 不能小于 18。",
				// "password": "password 不能为空!",
				// "confirm_password": "confirm_password 不能为空!",
				// "website_url": "website_url 不能为空!",
				// "role": "role 不能为空!",
				// "security_code": "security_code 不能为空!", // 如果 validate:"len=6" 前面没有 required, 空值不会触发len
				// "items_list": "items_list 不能为空!", // 同上
			},
		},
	}
	// 修正 multiple_errors 测试用例，使其更可预测
	// 让每个字段违反一个明确的规则，而不是一个字段违反多个规则
	multipleErrorsDTO := TestDTO{
		Username:        "us",                  // min=3
		Password:        "123",                 // min=6
		ConfirmPassword: "12345",             // eqfield=Password (假设Password是"123") -> 但 Password 也是错的
		Email:           "invalid",             // email
		Age:             5,                     // min=18
		Website:         "badurl",              // url
		Role:            "badrole",             // oneof
		Code:            "123",                 // len=6
		Items:           []string{},            // min=1 (如果前面有required, 这个可以不填)
		RetryAttempts:   10,                    // lte=5
		OldField:        "123",                 // nefield=Password
		Score:           200,                   // lt=100
		Bio:             strings.Repeat("b", 201), // max=200
	}
	// 重新获取 password 来设置 ConfirmPassword 和 OldField
	multipleErrorsDTO.Password = "short" // min=6
	multipleErrorsDTO.ConfirmPassword = "notequal" // eqfield=Password
	multipleErrorsDTO.OldField = "short" // nefield=Password

	expectedMultipleErrorsMap := map[string]string{
		"username":         "username 长度不能少于 3 个字符。",
		"password":         "password 长度不能少于 6 个字符。",
		"confirm_password": "confirm_password 必须等于 Password.",
		"email_field":      "email_field 必须是有效的邮箱地址。",
		"age":              "age 不能小于 18。",
		"website_url":      "website_url 必须是一个有效的URL。",
		"role":             "role 必须是 [admin, user, guest] 中的一个。",
		"security_code":    "security_code 长度必须是 6 个字符。",
		// "items_list": "items_list 不能少于 1 项。", // 如果是空切片且没有 required, min/max 不会触发
		"retry_attempts":   "retry_attempts 必须小于或等于 5.",
		"old_field":        "old_field 不能等于 Password.",
		"score":            "score 必须小于 100。",
		"biography":        "biography 长度不能超过 200 个字符。",
	}
	// 如果 items_list 有 required, 则会报错。当前 DTO 定义中没有 required。
	// 如果希望测试 items_list 的 min/max, 它的 validate 应该是 "required,min=1,max=5"
	// 或者在测试时提供一个非 nil 但不满足条件的 slice。
	// 例如，`Items: []string{}` 且 validate `min=1` 会报错。
	// 对于 `Items: []string{}` 和 `validate:"min=1,max=5"`，它违反了 min=1。
	// 我们在上面的 testCases 中已经有了 items_min_items 和 items_max_items。
	// 所以 multiple_errorsDTO 中的 Items: []string{} 应该触发 items_list 的 min 错误。
	// 让我们把 items_list 的错误也加入到 expectedMultipleErrorsMap。
	// 但是 TestDTO 的 items_list 已经是 `validate:"min=1,max=5"`，所以空切片 `[]string{}` 会触发 `min=1`
	expectedMultipleErrorsMap["items_list"] = "items_list 不能少于 1 项。"


	// 运行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validationErrs := getValidationErrors(tc.dtoInstance)
			if validationErrs == nil && (tc.expectedError != "" || tc.expectedMap != nil) {
				t.Fatalf("Expected validation errors, but got none for DTO: %+v", tc.dtoInstance)
			}
			if validationErrs != nil && tc.expectedError == "" && tc.expectedMap == nil {
				t.Fatalf("Got unexpected validation errors: %v", validationErrs)
			}

			translated := translator.TranslateValidationErrors(validationErrs)

			if tc.expectedMap != nil {
				assert.Equal(t, tc.expectedMap, translated, "Translated error map does not match expected map.")
			} else if tc.expectedError != "" {
				// 如果只期望一个错误，检查 map 是否只包含这一个错误，并且消息匹配
				if assert.Len(t, translated, 1, "Expected a single error in the map.") {
					// 查找 map 中的第一个 (也应该是唯一一个) 错误
					var actualMsg string
					var actualField string
					for field, msg := range translated {
						actualField = field // json tag name
						actualMsg = msg
						break
					}
					// 提取预期错误中的字段名 (json tag)
					// 预期错误格式: "json_field_name: 错误消息"
					// 这里简化，假设 tc.expectedError 就是消息部分，字段名从 tc.dtoInstance 和 tag 中推断
					// 或者，让 expectedError 的格式为 "json_field_name: message"
					// 当前的 testCases.expectedMap 更好，我们只使用 expectedMap
					// 如果 tc.expectedError 存在，它应该等同于 tc.expectedMap 的唯一值
					assert.Contains(t, actualMsg, tc.expectedError,
						fmt.Sprintf("Field '%s' error message '%s' does not contain expected string '%s'",
							actualField, actualMsg, tc.expectedError))
				}
			}
		})
	}

	// 单独测试 multiple_errorsDTO
	t.Run("multiple_errors_scenario", func(t *testing.T) {
		validationErrs := getValidationErrors(&multipleErrorsDTO)
		if validationErrs == nil {
			t.Fatal("Expected validation errors for multiple_errorsDTO, but got none.")
		}
		translated := translator.TranslateValidationErrors(validationErrs)
		assert.Equal(t, expectedMultipleErrorsMap, translated, "Translated error map for multiple_errorsDTO does not match expected map.")
	})

	// 测试 TranslateValidationErrors 使用 nil 输入
	t.Run("nil_validation_errors", func(t *testing.T) {
		translated := translator.TranslateValidationErrors(nil)
		assert.Empty(t, translated, "Expected empty map for nil validation errors.")
	})

	// 测试一个字段违反多个规则的情况 (validator 通常只报告第一个)
	// 例如，Username `validate:"required,alphanum,min=3,max=20"`
	// 如果 Username 为 "u!"
	t.Run("username_violates_alphanum_and_min", func(t *testing.T) {
		dto := TestDTO{
			Username:        "u!", // 违反 alphanum 和 min=3
			Password:        "validpass",
			ConfirmPassword: "validpass",
			Email:           "test@example.com",
			Website:         "http://example.com",
			Role:            "admin",
			Code:            "123456",
			Items:           []string{"one"},
		}
		validationErrs := getValidationErrors(&dto)
		translated := translator.TranslateValidationErrors(validationErrs)
		// Validator v10 按声明顺序处理标签，或者有自己的优先级。
		// "alphanum" 通常先于 "min" 被检查。
		expected := map[string]string{"username": "username 只能包含字母和数字。"}
		assert.Equal(t, expected, translated)
	})

	// 测试 RegisterTagNameFunc 是否正确使用了 json 标签
	t.Run("field_name_uses_json_tag", func(t *testing.T) {
		dto := TestDTO{Email: "not-an-email"} // Email 字段的 json tag 是 "email_field"
		// 需要填充其他必填字段以隔离测试
		dto.Username = "validuser"
		dto.Password = "validpass"
		dto.ConfirmPassword = "validpass"
		dto.Website = "http://example.com"
		dto.Role = "admin"
		dto.Code = "123456"
		dto.Items = []string{"item1"}

		validationErrs := getValidationErrors(&dto)
		translated := translator.TranslateValidationErrors(validationErrs)
		expected := map[string]string{"email_field": "email_field 必须是有效的邮箱地址。"}
		assert.Equal(t, expected, translated, "Field name in error map should be the json tag.")
	})

	// 测试 eqfield 中参数字段的名称
	t.Run("eqfield_param_name", func(t *testing.T){
		dto := TestDTO{
			Username: "validuser",
			Password: "password123",
			ConfirmPassword: "password456", // 不等于 Password
			Email: "test@example.com",
			Website: "http://example.com",
			Role: "admin",
			Code: "123456",
			Items: []string{"one"},
		}
		validationErrs := getValidationErrors(&dto)
		translated := translator.TranslateValidationErrors(validationErrs)
		// 我们的翻译是 "{0} 必须等于 {1}."，其中 {1} 是 StructFieldName
		// 即 Password (结构体字段名), 而不是 password (json tag)
		expected := map[string]string{"confirm_password": "confirm_password 必须等于 Password."}
		assert.Equal(t, expected, translated)
	})

	// 测试 oneof 翻译中的参数格式
	t.Run("oneof_param_format", func(t *testing.T){
		dto := TestDTO{
			Username: "validuser",
			Password: "password123",
			ConfirmPassword: "password123",
			Email: "test@example.com",
			Website: "http://example.com",
			Role: "invalid_role", // 不在 "admin user guest" 中
			Code: "123456",
			Items: []string{"one"},
		}
		validationErrs := getValidationErrors(&dto)
		translated := translator.TranslateValidationErrors(validationErrs)
		// 我们的翻译是 "{0} 必须是 [{1}] 中的一个。" {1} 是 "admin, user, guest"
		expected := map[string]string{"role": "role 必须是 [admin, user, guest] 中的一个。"}
		assert.Equal(t, expected, translated)
	})

}

// TestRegisterTagNameFunc_IgnoreField 测试 RegisterTagNameFunc 对 json:"-" 的处理
func TestRegisterTagNameFunc_IgnoreField(t *testing.T) {
	type DTOWithIgnoredField struct {
		VisibleField string `json:"visible_field" validate:"required"`
		HiddenField  string `json:"-" validate:"required"` // 这个 required 不应该被触发并包含在错误中
	}

	// 初始化一个新的 validator 实例，并重新配置 TagNameFunc
	// 因为全局的 validator 实例在 init() 中已经配置过了
	// 为了隔离测试 TagNameFunc 的特定行为，最好在这里用一个新的 validator
	// 但 InitTranslator() 修改的是 binding.Validator.Engine()，是全局的
	// 所以我们依赖 init() 中对全局 validator 的设置

	// dto := DTOWithIgnoredField{VisibleField: ""} // HiddenField 也为空
	// validate := validator.New() // 使用新的 validator
	// // 手动调用 InitTranslator 的一部分来注册 TagNameFunc (或者确保全局的被正确设置)
	// // 这部分比较难独立测试，因为它修改了全局状态。
	// // 假设 InitTranslator() 已经正确运行。

	// err := validate.Struct(dto)
	// validationErrs, _ := err.(validator.ValidationErrors)
	// translated := translator.TranslateValidationErrors(validationErrs)

	// // 期望只看到 visible_field 的错误
	// expected := map[string]string{"visible_field": "visible_field 不能为空!"}
	// assert.Equal(t, expected, translated)

	// 上述测试方法存在问题，因为 InitTranslator 修改的是 gin 全局 validator
	// 而我们在这里用 validator.New() 创建的是新的。
	// 正确的测试方法是，确保 InitTranslator 被调用，然后使用一个会通过 gin validator 的方式
	// 但这里是单元测试 translator 包，不应该依赖 gin。

	// 更好的方式是：检查 TranslateValidationErrors 的行为。
	// 如果一个字段的 FieldError.Field() 返回空字符串（因为 TagNameFunc 返回了空），
	// 那么 TranslateValidationErrors 应该如何处理？目前会用空字符串作为 key。
	// 这通常不是问题，因为 validator 不会为 `json:"-"` 的字段产生错误（除非自定义）。
	// validator/v10 本身在处理 Struct 时，如果 RegisterTagNameFunc 返回空字符串，
	// 它会跳过该字段的验证。所以 HiddenField 上的 "required" 不会被触发。

	dto := struct {
		VisibleField string `json:"visible_field" validate:"required"`
		HiddenField  string `json:"-" validate:"required"` // 这个 required 不应该被触发
	}{VisibleField: ""}

	// validate := validator.New() // 获取一个 validator 实例
	// 如果要测试全局 validator 的 TagNameFunc, 需要通过 binding.Validator.Engine()
	vEngine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatal("Could not get gin's validator engine")
	}


	err := vEngine.Struct(dto) // 使用 gin 的 validator 引擎
	validationErrs, _ := err.(validator.ValidationErrors)
	translated := translator.TranslateValidationErrors(validationErrs)

	// 期望只看到 visible_field 的错误
	expected := map[string]string{"visible_field": "visible_field 不能为空!"}
	assert.Equal(t, expected, translated, "Error from field with json:\"-\" should be ignored by validator due to TagNameFunc.")
}
