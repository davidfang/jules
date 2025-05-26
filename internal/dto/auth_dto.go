// Package dto (Data Transfer Object) 定义了用于在不同层之间传输数据的结构体。
package dto

// UserLoginReq 代表用户登录请求的数据结构。
// UsernameOrEmail: 用户名或电子邮件地址。
// Password: 用户密码。
type UserLoginReq struct {
	// UsernameOrEmail 是用户用于登录的标识，可以是用户名或电子邮件地址。
	// `json:"username_or_email"`: 在 JSON 中此字段名为 "username_or_email"。
	// `binding:"required"`: 此字段为必填项。
	UsernameOrEmail string `json:"username_or_email" binding:"required"`

	// Password 是用户的登录密码。
	// `json:"password"`: 在 JSON 中此字段名为 "password"。
	// `binding:"required"`: 此字段为必填项。
	Password string `json:"password" binding:"required"`
}

// UserLoginRes 代表用户成功登录后返回的数据结构。
// AccessToken: 生成的 JWT 访问令牌。
// TokenType: 令牌类型，通常为 "Bearer"。
// ExpiresIn: 令牌的有效期截止时间戳 (Unix timestamp)。
type UserLoginRes struct {
	// AccessToken 是用户成功登录后获取到的 JWT。
	// 客户端在后续请求中应将此 Token 包含在 Authorization 请求头中 (通常以 "Bearer " 为前缀)。
	AccessToken string `json:"access_token"`

	// TokenType 表示令牌的类型。对于 JWT，这通常是 "Bearer"。
	TokenType string `json:"token_type"`

	// ExpiresIn 表示 AccessToken 的过期时间戳 (Unix timestamp in seconds)。
	// 客户端可以使用此信息来管理 Token 的生命周期，例如在过期前刷新 Token (如果支持刷新令牌)。
	ExpiresIn int64 `json:"expires_in"`

	// UserID 是成功登录的用户的唯一标识符。
	// 可选地在登录响应中返回用户ID，便于前端使用。
	UserID uint `json:"user_id,omitempty"`

	// Username 是成功登录的用户的用户名。
	// 可选地在登录响应中返回用户名，便于前端使用。
	Username string `json:"username,omitempty"`
}

// RefreshTokenReq (可选) 代表刷新令牌的请求。
// type RefreshTokenReq struct {
//	 RefreshToken string `json:"refresh_token" binding:"required"`
// }

// RefreshTokenRes (可选) 代表刷新令牌的响应。
// type RefreshTokenRes struct {
//	 AccessToken string `json:"access_token"`
//	 TokenType   string `json:"token_type"`
//	 ExpiresIn   int64  `json:"expires_in"`
// }
