// Package jwtutil 提供了生成和解析 JWT (JSON Web Tokens) 的工具函数。
// 它使用了 github.com/golang-jwt/jwt/v5 库。
package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"errors" // 确保 errors 包已导入
)

// CustomClaims 结构体定义了 JWT 中自定义的声明字段。
// 它嵌入了 jwt.RegisteredClaims，这包含了标准的 JWT 声明 (如 iss, sub, exp, nbf, iat, jti)。
// UserID: 用户的唯一标识符。
// Username: 用户的登录名。
type CustomClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 函数根据给定的用户 ID、用户名和 Viper 配置生成一个新的 JWT。
// userID: 要包含在 Token 中的用户 ID。
// username: 要包含在 Token 中的用户名。
// cfg: Viper 配置实例，用于读取 JWT 相关的设置 (secret_key, issuer, access_token_expire_duration)。
// 返回:
//   - tokenString (string): 生成的 JWT 字符串。
//   - expiresAt (int64): Token 的过期时间戳 (Unix timestamp in seconds)。
//   - err (error): 如果生成过程中发生错误，则返回错误。
func GenerateToken(userID uint, username string, cfg *viper.Viper) (tokenString string, expiresAt int64, err error) {
	// 1. 从配置中获取 JWT 相关设置
	secretKey := cfg.GetString("jwt.secret_key")
	if secretKey == "" {
		return "", 0, fmt.Errorf("JWT 密钥 (jwt.secret_key) 未在配置中设置")
	}
	issuer := cfg.GetString("jwt.issuer")
	if issuer == "" {
		issuer = "go-base-system-v2" // 默认签发者
	}
	expireDurationStr := cfg.GetString("jwt.access_token_expire_duration")
	if expireDurationStr == "" {
		expireDurationStr = "1h" // 默认过期时间为 1 小时
	}
	expireDuration, err := time.ParseDuration(expireDurationStr)
	if err != nil {
		return "", 0, fmt.Errorf("解析 JWT 过期时间 '%s' 失败: %w", expireDurationStr, err)
	}

	// 2. 计算过期时间
	expirationTime := time.Now().Add(expireDuration)
	expiresAt = expirationTime.Unix() // 获取 Unix 时间戳 (秒)

	// 3. 创建自定义声明 (CustomClaims)
	claims := &CustomClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime), // 设置过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),     // 设置签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),     // 设置生效时间 (通常与签发时间相同)
			Issuer:    issuer,                             // 设置签发者
			Subject:   fmt.Sprintf("%d", userID),          // 主题，通常是用户ID的字符串表示
			// ID:        uuid.NewString(),                // 可选：为 Token 设置唯一ID (JTI)
		},
	}

	// 4. 使用 HS256 签名算法和声明创建 Token 对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 5. 使用密钥对 Token 对象进行签名，生成 JWT 字符串
	tokenString, err = token.SignedString([]byte(secretKey))
	if err != nil {
		return "", 0, fmt.Errorf("JWT 签名失败: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ParseToken 函数解析并验证给定的 JWT 字符串。
// tokenString: 要解析的 JWT 字符串。
// cfg: Viper 配置实例，用于读取 JWT 密钥 (jwt.secret_key)。
// 返回:
//   - *CustomClaims: 如果 Token 有效且解析成功，则返回包含自定义声明的指针。
//   - error: 如果 Token 无效、解析失败或验证失败，则返回错误。
func ParseToken(tokenString string, cfg *viper.Viper) (*CustomClaims, error) {
	// 1. 从配置中获取 JWT 密钥
	secretKey := cfg.GetString("jwt.secret_key")
	if secretKey == "" {
		return nil, fmt.Errorf("JWT 密钥 (jwt.secret_key) 未在配置中设置，无法解析 Token")
	}

	// 2. 解析 Token
	// jwt.ParseWithClaims 函数会解析 Token 字符串，并使用提供的密钥函数 (keyFunc) 来获取验证签名所需的密钥。
	// 它还会验证 Token 的标准声明 (例如过期时间 exp, 生效时间 nbf, 签发时间 iat)。
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 确保 Token 使用的签名算法与我们期望的算法 (HS256) 一致。
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil // 返回用于验证签名的密钥
	})

	if err != nil {
		// 处理不同类型的解析错误
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, fmt.Errorf("Token 格式错误: %w", err)
		} else if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("Token 已过期: %w", err)
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, fmt.Errorf("Token 尚未生效: %w", err)
		}
		// 其他解析或验证错误
		return nil, fmt.Errorf("Token 解析或验证失败: %w", err)
	}

	// 3. 检查 Token 是否有效，并提取声明
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		// Token 有效且声明成功提取
		return claims, nil
	}

	return nil, fmt.Errorf("无效的 Token (声明类型不匹配或 Token.Valid 为 false)")
}

// 确保 errors 包已在文件顶部导入。 // 此注释可以移除，因为上面已经添加了导入
