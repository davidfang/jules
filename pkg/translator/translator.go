// Package translator 提供了校验错误信息中文翻译的功能。
package translator

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

// Trans 是一个全局的翻译器实例，用于将校验错误信息翻译成中文。
var Trans ut.Translator

// InitTranslator 初始化翻译器。
// 它会设置中文为默认语言，并为 Gin 的默认校验器注册中文翻译和自定义字段名处理。
func InitTranslator() error {
	// 1. 初始化中文语言环境
	zhLocale := zh.New()

	// 2. 创建通用翻译器，设置中文为默认和备选语言
	uni := ut.New(zhLocale, zhLocale)

	// 3. 从通用翻译器获取中文翻译器实例
	var found bool
	Trans, found = uni.GetTranslator("zh")
	if !found {
		return &TranslatorInitializationError{message: "translator not found: zh"}
	}

	// 4. 获取 Gin 的默认校验器引擎
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil // 如果不是 *validator.Validate 类型，则不进行后续操作
	}

	// 5. 注册字段名获取函数，优先使用 json 标签作为字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" { // 如果 json 标签是 "-", 则忽略该字段
			return ""
		}
		// 也可以在这里添加对其他标签（如 "comment"）的处理，以提供更友好的字段名
		// comment := fld.Tag.Get("comment")
		// if comment != "" {
		//	   return comment
		// }
		return name // 默认返回 json 标签名
	})

	// 6. 为校验器注册中文翻译器
	if err := zhTranslations.RegisterDefaultTranslations(v, Trans); err != nil {
		return err
	}

	// 7. 注册自定义的错误消息翻译
	registerCustomTranslations(v)

	return nil
}

// registerCustomTranslations 为特定的校验规则注册自定义的中文翻译。
func registerCustomTranslations(v *validator.Validate) {
	// 示例：为 'required' 规则注册翻译
	v.RegisterTranslation("required", Trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} 不能为空!", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field()) // fe.Field() 会是 RegisterTagNameFunc 返回的值
		return t
	})

	// 为 'min' 规则注册翻译 (需要区分类型：字符串、数字、数组/切片)
	v.RegisterTranslation("min", Trans, func(ut ut.Translator) error {
		if err := ut.Add("min-string", "{0} 长度不能少于 {1} 个字符。", true); err != nil {
			return err
		}
		if err := ut.Add("min-number", "{0} 不能小于 {1}。", true); err != nil {
			return err
		}
		if err := ut.Add("min-items", "{0} 不能少于 {1} 项。", true); err != nil {
			return err
		}
		return nil
	}, func(ut ut.Translator, fe validator.FieldError) string {
		var t string
		var err error
		param := fe.Param()
		switch fe.Kind() {
		case reflect.String:
			t, err = ut.T("min-string", fe.Field(), param)
		case reflect.Slice, reflect.Array, reflect.Map:
			t, err = ut.T("min-items", fe.Field(), param)
		default: // 默认为数字类型
			t, err = ut.T("min-number", fe.Field(), param)
		}
		if err != nil {
			// 理论上不应发生，因为 key 已经 Add 过了
			return fe.(error).Error()
		}
		return t
	})

	// 为 'max' 规则注册翻译 (需要区分类型：字符串、数字、数组/切片)
	v.RegisterTranslation("max", Trans, func(ut ut.Translator) error {
		if err := ut.Add("max-string", "{0} 长度不能超过 {1} 个字符。", true); err != nil {
			return err
		}
		if err := ut.Add("max-number", "{0} 不能大于 {1}。", true); err != nil {
			return err
		}
		if err := ut.Add("max-items", "{0} 不能超过 {1} 项。", true); err != nil {
			return err
		}
		return nil
	}, func(ut ut.Translator, fe validator.FieldError) string {
		var t string
		var err error
		param := fe.Param()
		switch fe.Kind() {
		case reflect.String:
			t, err = ut.T("max-string", fe.Field(), param)
		case reflect.Slice, reflect.Array, reflect.Map:
			t, err = ut.T("max-items", fe.Field(), param)
		default: // 默认为数字类型
			t, err = ut.T("max-number", fe.Field(), param)
		}
		if err != nil {
			return fe.(error).Error()
		}
		return t
	})

	// 为 'len' 规则注册翻译 (需要区分类型：字符串、数字、数组/切片)
	// 注意：validator 的 'len' 规则通常用于字符串和集合的长度，数字的精确值比较用 'eq'
	v.RegisterTranslation("len", Trans, func(ut ut.Translator) error {
		if err := ut.Add("len-string", "{0} 长度必须是 {1} 个字符。", true); err != nil {
			return err
		}
		if err := ut.Add("len-items", "{0} 必须包含 {1} 项。", true); err != nil {
			return err
		}
		// 'len' 不常用于数字的直接比较，但如果需要可以添加
		// if err := ut.Add("len-number", "{0} 必须等于 {1}。", true); err != nil {
		// 	return err
		// }
		return nil
	}, func(ut ut.Translator, fe validator.FieldError) string {
		var t string
		var err error
		param := fe.Param()
		switch fe.Kind() {
		case reflect.String:
			t, err = ut.T("len-string", fe.Field(), param)
		case reflect.Slice, reflect.Array, reflect.Map:
			t, err = ut.T("len-items", fe.Field(), param)
		// default:
		// 	t, err = ut.T("len-number", fe.Field(), param)
		}
		if err != nil {
			return fe.(error).Error()
		}
		return t
	})

	// 为 'email' 规则注册翻译
	v.RegisterTranslation("email", Trans, func(ut ut.Translator) error {
		return ut.Add("email", "{0} 必须是有效的邮箱地址。", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("email", fe.Field())
		return t
	})

	// 为 'eqfield' 规则注册翻译
	v.RegisterTranslation("eqfield", Trans, func(ut ut.Translator) error {
		return ut.Add("eqfield", "{0} 必须等于 {1}.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		// eqfield 的参数是另一个字段的名称，需要特殊处理一下，从 FieldError 中获取比较字段的名称
		// fe.Param() 返回的是比较字段的名称，我们需要用它来替换 {1}
		// fe.Field() 是当前字段的名称
		// Gin 的 validator 默认的 eqfield 翻译是 "{0} must be equal to {1}"
		// 我们需要获取比较字段的 "json" tag 或者 "comment" tag
		// 这里简化处理，直接使用 fe.Param()，它会是结构体内的字段名
		// 如果希望 {1} 也是 json tag 名，需要更复杂的处理，在 RegisterTagNameFunc 中缓存所有字段的 json tag
		// 或者在 DTO 定义时，确保 eqfield 的参数也是 json tag 名（但这不符合 validator 的设计）
		// 目前的实现，如果 User struct 中有 Password string `json:"password"` 和 ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
		// 错误会是 "confirm_password 必须等于 Password."
		// 如果希望是 "confirm_password 必须等于 password." (json tag of Password field), 需要额外逻辑
		t, _ := ut.T("eqfield", fe.Field(), fe.Param())
		return t
	})

	// 为 'nefield' 规则注册翻译
	v.RegisterTranslation("nefield", Trans, func(ut ut.Translator) error {
		return ut.Add("nefield", "{0} 不能等于 {1}.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("nefield", fe.Field(), fe.Param())
		return t
	})

	// 为 'gt' 规则注册翻译 (通常用于数字，也可以用于字符串长度等，但我们主要关注数字)
	// validator v10 默认的 gt 翻译已经区分了类型 (Numeric, String, Slice, Map)
	// 这里我们提供一个统一的数字比较翻译，如果需要细分，可以像 min/max 一样处理
	v.RegisterTranslation("gt", Trans, func(ut ut.Translator) error {
		return ut.Add("gt", "{0} 必须大于 {1}.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("gt", fe.Field(), fe.Param())
		return t
	})

	// 为 'gte' 规则注册翻译
	v.RegisterTranslation("gte", Trans, func(ut ut.Translator) error {
		return ut.Add("gte", "{0} 必须大于或等于 {1}.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("gte", fe.Field(), fe.Param())
		return t
	})

	// 为 'lt' 规则注册翻译
	v.RegisterTranslation("lt", Trans, func(ut ut.Translator) error {
		return ut.Add("lt", "{0} 必须小于 {1}.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("lt", fe.Field(), fe.Param())
		return t
	})

	// 为 'lte' 规则注册翻译
	v.RegisterTranslation("lte", Trans, func(ut ut.Translator) error {
		return ut.Add("lte", "{0} 必须小于或等于 {1}.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("lte", fe.Field(), fe.Param())
		return t
	})

	// 为 'alphanum' 规则注册翻译
	v.RegisterTranslation("alphanum", Trans, func(ut ut.Translator) error {
		return ut.Add("alphanum", "{0} 只能包含字母和数字。", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("alphanum", fe.Field())
		return t
	})

	// 为 'url' 规则注册翻译
	v.RegisterTranslation("url", Trans, func(ut ut.Translator) error {
		return ut.Add("url", "{0} 必须是一个有效的URL。", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("url", fe.Field())
		return t
	})

	// 为 'oneof' 规则注册翻译
	// fe.Param() 返回的是 'value1 value2 value3' 这样的字符串
	v.RegisterTranslation("oneof", Trans, func(ut ut.Translator) error {
		return ut.Add("oneof", "{0} _PLACEHOLDER_FOR_ONEOF_MSG_ {1}.", true) // 占位符将在下面替换
	}, func(ut ut.Translator, fe validator.FieldError) string {
		// 将 fe.Param() 中的空格替换为逗号，以符合 "必须是 [value1, value2] 中的一个" 格式
		param := strings.ReplaceAll(fe.Param(), " ", ", ")
		// 使用一个临时的 key 来格式化 oneof 消息，因为 Add 的时候不能直接用 fe.Param()
		// 更好的做法是 Add 一个通用的模板，例如 "{0} must be one of [{1}]"
		// 然后在翻译函数中替换 {1}
		// 这里我们直接构造消息，或者使用一个更通用的模板
		// t, _ := ut.T("oneof", fe.Field(), param) // 这样 {1} 会是整个 "value1, value2"
		// 我们希望的是 "字段 必须是 [value1, value2] 中的一个"
		// 所以，在 Add 中，我们使用一个特定的key，或者直接在翻译函数中构建
		// 让我们重新定义 "oneof" 的模板
		
		// 先移除之前可能添加的 "oneof" (如果 Add 的 key 相同，新的会覆盖旧的)
		// 这里我们直接使用 fe.Field() 和 param 来构造消息，不依赖于 Add 中的复杂模板
		// 这是因为 ut.T 的参数是 fe.Field() (对应 {0}) 和 fe.Param() (对应 {1})
		// 而我们需要的是 "{0} 必须是 [{1}] 中的一个"
		// 所以，我们直接用 fe.Field() 和处理过的 param 构造
		// 或者，我们可以在 Add 中用一个更简单的模板，例如: "{0} 必须在给定选项中。" 然后在这里附加选项。
		// 为了简单起见，我们直接构造。
		// 更好的方式是 ut.Add("oneof", "{0} must be one of [{1}]", true)
		// 然后 t, _ := ut.T("oneof", fe.Field(), param)

		// 修正 oneof 的注册方式
		// 1. Add 时使用一个更标准的模板
		// 2. T 的时候传入处理好的 param
		// 我们在上面 Add 的时候用了一个特殊的占位符，这里我们不用那个了。
		// 假设我们已经 Add 了一个通用的 "oneof_custom" 模板
		_ = ut.Add("oneof_custom_msg_key", "{0} 必须是 [{1}] 中的一个。", true) // 确保这个 key 是唯一的
		t, _ := ut.T("oneof_custom_msg_key", fe.Field(), param)
		return t
	})
}

// TranslateValidationErrors 将 validator.ValidationErrors 翻译成中文的错误消息映射。
// errs: Gin 校验器返回的 ValidationErrors。
// 返回: 一个 map，键是字段名 (通常是 JSON 标签名)，值是翻译后的错误消息。
// 如果 errs 为 nil 或空，则返回一个空的 map。
func TranslateValidationErrors(errs validator.ValidationErrors) map[string]string {
	result := make(map[string]string)
	if errs == nil {
		return result
	}

	for _, err := range errs {
		// fe.Field() 应该返回 RegisterTagNameFunc 中处理过的字段名
		// fe.Translate(Trans) 返回翻译后的错误信息
		result[err.Field()] = err.Translate(Trans)
	}
	return result
}

// Placeholder for a real error type if needed in InitTranslator
type TranslatorInitializationError struct {
    message string
}

func (e *TranslatorInitializationError) Error() string {
    return e.message
}
