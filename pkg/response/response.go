// Package response 提供统一的 API 响应处理功能。
// 它定义了标准的 JSON 响应结构和便捷的辅助函数，用于生成成功或失败的响应。
package response

import (
	"net/http" // 导入 net/http 包，用于 HTTP 状态码常量。

	"github.com/gin-gonic/gin" // 导入 Gin 框架包。
)

// ResponseData 定义了 API 响应的标准数据结构。
// Code: 业务状态码，用于表示具体的业务处理结果（例如，0 代表成功，其他数字代表不同类型的错误）。
// Msg:  响应消息，通常用于提供给用户或前端的提示信息。
// Data: 实际的响应数据内容，可以是任何类型 (interface{})。
type ResponseData struct {
	Code int         `json:"code"` // 业务状态码
	Msg  string      `json:"msg"`  // 响应消息
	Data interface{} `json:"data"` // 响应数据
}

// DefaultBizErrorCode 是用于 FailWithMessage 函数的默认业务错误码。
// 当业务层没有指定具体的错误码时，可以使用此默认值。
const DefaultBizErrorCode = -1

// Success 函数用于生成并发送一个成功的 API 响应。
// 它使用 HTTP 状态码通常为 200 OK (或 2xx 系列)，并将业务状态码 Code 设置为 0。
// c: Gin 的上下文对象，用于发送响应。
// httpStatusCode: HTTP 状态码 (例如 http.StatusOK)。
// data: 要包含在响应中的数据。
func Success(c *gin.Context, httpStatusCode int, data interface{}) {
	c.JSON(httpStatusCode, ResponseData{
		Code: 0,    // 0 通常表示业务处理成功
		Msg:  "success", // 成功的默认消息
		Data: data,
	})
}

// Fail 函数用于生成并发送一个失败的 API 响应。
// 它允许指定 HTTP 状态码、业务错误码和错误消息。
// c: Gin 的上下文对象。
// httpStatusCode: HTTP 状态码 (例如 http.StatusBadRequest, http.StatusInternalServerError)。
// bizCode: 自定义的业务错误码，用于更细致地区分错误类型。
// msg: 具体的错误消息。
func Fail(c *gin.Context, httpStatusCode int, bizCode int, msg string) {
	c.JSON(httpStatusCode, ResponseData{
		Code: bizCode,
		Msg:  msg,
		Data: map[string]interface{}{}, // 当没有具体错误数据时，Data 字段返回一个空 JSON 对象
	})
}

// FailWithMessage 函数用于生成并发送一个失败的 API 响应，使用默认的业务错误码。
// 当只需要提供错误消息而不需要指定特定业务错误码时，可以使用此函数。
// c: Gin 的上下文对象。
// httpStatusCode: HTTP 状态码。
// msg: 具体的错误消息。
func FailWithMessage(c *gin.Context, httpStatusCode int, msg string) {
	c.JSON(httpStatusCode, ResponseData{
		Code: DefaultBizErrorCode, // 使用预定义的默认业务错误码
		Msg:  msg,
		Data: map[string]interface{}{}, // 当没有具体错误数据时，Data 字段返回一个空 JSON 对象
	})
}

// FailWithData 函数用于生成并发送一个失败的 API 响应，同时可以携带一些额外数据。
// 例如，在参数校验失败时，可以将校验失败的字段信息放在 Data 中。
// c: Gin 的上下文对象。
// httpStatusCode: HTTP 状态码。
// bizCode: 自定义的业务错误码。
// msg: 具体的错误消息。
// data: 附加的错误相关数据。
func FailWithData(c *gin.Context, httpStatusCode int, bizCode int, msg string, data interface{}) {
	c.JSON(httpStatusCode, ResponseData{
		Code: bizCode,
		Msg:  msg,
		Data: data,
	})
}

// FailWithValidationErrors 函数用于统一处理参数校验错误。
// 它返回一个包含详细错误信息的 JSON 响应，其中 data 字段是一个 map，列出校验失败的字段及其错误信息。
// c: Gin 的上下文对象。
// bizCode: 自定义的业务错误码，用于标识参数校验错误类型。
// msg: 主错误消息，例如 "输入参数校验失败"。
// details: 一个 map[string]string，键是校验失败的字段名，值是该字段具体的错误描述。
func FailWithValidationErrors(c *gin.Context, bizCode int, msg string, details map[string]string) {
	c.JSON(http.StatusBadRequest, ResponseData{ // 参数校验错误通常使用 HTTP 400 Bad Request
		Code: bizCode,
		Msg:  msg,
		Data: details, // data 字段包含详细的校验错误信息
	})
}

// SuccessOK 是一个便捷函数，直接发送 HTTP 200 OK 的成功响应。
// data: 要包含在响应中的数据。
func SuccessOK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, data)
}

// FailBadRequest 是一个便捷函数，直接发送 HTTP 400 Bad Request 的失败响应。
// msg: 具体的错误消息。
func FailBadRequest(c *gin.Context, msg string) {
	FailWithMessage(c, http.StatusBadRequest, msg)
}

// FailUnauthorized 是一个便捷函数，直接发送 HTTP 401 Unauthorized 的失败响应。
// msg: 具体的错误消息。
func FailUnauthorized(c *gin.Context, msg string) {
	FailWithMessage(c, http.StatusUnauthorized, msg)
}

// FailForbidden 是一个便捷函数，直接发送 HTTP 403 Forbidden 的失败响应。
// msg: 具体的错误消息。
func FailForbidden(c *gin.Context, msg string) {
	FailWithMessage(c, http.StatusForbidden, msg)
}

// FailNotFound 是一个便捷函数，直接发送 HTTP 404 Not Found 的失败响应。
// msg: 具体的错误消息。
func FailNotFound(c *gin.Context, msg string) {
	FailWithMessage(c, http.StatusNotFound, msg)
}

// FailInternalError 是一个便捷函数，直接发送 HTTP 500 Internal Server Error 的失败响应。
// msg: 具体的错误消息。
func FailInternalError(c *gin.Context, msg string) {
	FailWithMessage(c, http.StatusInternalServerError, msg)
}
