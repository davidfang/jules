// Package dto (Data Transfer Object) 定义了用于在不同层之间传输数据的结构体。
// 这些结构体通常用于 API 请求/响应、服务层方法参数等。
package dto

// UserRegisterReq 代表用户注册请求的数据结构。
// 它包含了用户注册时需要提供的基本信息：用户名、电子邮件和密码。
// 使用了 `json` 标签来指定在 JSON 序列化/反序列化时使用的字段名。
// 使用了 `binding` 标签来配合 Gin 等框架进行请求数据的校验，例如 `binding:"required"`。
type UserRegisterReq struct {
	// Username 是用户期望的登录名。
	// `json:"username"`: 在 JSON 中此字段名为 "username"。
	// `binding:"required,min=3,max=50"`: 此字段为必填项，长度必须在 3 到 50 个字符之间。
	Username string `json:"username" binding:"required,min=3,max=50"`

	// Email 是用户的电子邮件地址。
	// `json:"email"`: 在 JSON 中此字段名为 "email"。
	// `binding:"required,email"`: 此字段为必填项，并且必须符合电子邮件的格式。
	Email string `json:"email" binding:"required,email"`

	// Password 是用户设置的密码。
	// `json:"password"`: 在 JSON 中此字段名为 "password"。
	// `binding:"required,min=8,max=100"`: 此字段为必填项，长度必须在 8 到 100 个字符之间。
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// UserLoginReq 代表用户登录请求的数据结构。
// （可以根据需要在此文件中添加更多 DTO 结构体）
// type UserLoginReq struct {
// 	Username string `json:"username" binding:"required"`
// 	Password string `json:"password" binding:"required"`
// }

// UserLoginRes 代表用户登录响应的数据结构。
// type UserLoginRes struct {
// 	Token    string `json:"token"`
// 	UserID   uint   `json:"user_id"`
// 	Username string `json:"username"`
// }
