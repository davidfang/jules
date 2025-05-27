// Package middleware 提供了 Gin HTTP 中间件。
// 中间件可以在请求处理链中的特定点执行代码，例如认证、日志、CORS 等。
package middleware

import (
	// HTTP 状态码常量
	"fmt"
	"go-base-system/pkg/jwtutil"  // JWT 工具包，用于解析 Token
	"go-base-system/pkg/response" // 统一 API 响应包
	"strings"                     // 用于字符串操作，例如处理 Authorization 请求头

	"github.com/casbin/casbin/v2" // 导入 Casbin 包

	"github.com/gin-gonic/gin" // Gin Web 框架
	"github.com/spf13/viper"   // Viper 配置管理库
	"go.uber.org/zap"          // Zap 高性能日志库
)

const (
	// AuthorizationHeaderKey 是 HTTP 请求头中用于传递认证令牌的键名。
	AuthorizationHeaderKey = "Authorization"
	// AuthorizationTypeBearer 是 Bearer Token 认证方案的类型标识。
	AuthorizationTypeBearer = "Bearer"
	// UserIDKey 是在 Gin Context 中存储用户 ID 的键名。
	UserIDKey = "userID"
	// UsernameKey 是在 Gin Context 中存储用户名的键名。
	UsernameKey = "username"
)

// JWTAuthMiddleware 是一个 Gin 中间件，用于 JWT (JSON Web Token) 认证。
// 它从 "Authorization" 请求头中提取 Bearer Token，验证 Token 的有效性，
// 如果 Token 有效，则将解析出的用户 ID 和用户名设置到 Gin 上下文中，以便后续 Handler 使用。
// 如果 Token 无效或缺失，则中止请求并返回 HTTP 401 Unauthorized 错误。
//
// 参数:
//
//	cfg: Viper 配置实例，用于读取 JWT 密钥等配置。
//	logger: Zap SugaredLogger 实例，用于记录认证过程中的信息和错误。
//
// 返回:
//
//	gin.HandlerFunc: Gin 处理函数，可注册为中间件。
func JWTAuthMiddleware(cfg *viper.Viper, logger *zap.SugaredLogger) gin.HandlerFunc {
	if cfg == nil {
		panic("JWTAuthMiddleware: Viper config instance is nil")
	}
	if logger == nil {
		panic("JWTAuthMiddleware: Zap SugaredLogger instance is nil")
	}

	return func(c *gin.Context) {
		// 1. 从 Authorization 请求头获取认证信息
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			logger.Warn("Authorization header is missing")
			response.FailUnauthorized(c, "请求未携带认证 Token")
			c.Abort() // 中止请求处理链
			return
		}

		// 2. 校验认证信息格式是否为 "Bearer <token>"
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != AuthorizationTypeBearer {
			logger.Warnw("Invalid Authorization header format", "header", authHeader)
			response.FailUnauthorized(c, "认证 Token 格式错误，应为 'Bearer <token>'")
			c.Abort()
			return
		}

		// 3. 提取 Token 字符串
		tokenString := headerParts[1]
		if tokenString == "" {
			logger.Warn("Bearer token is empty")
			response.FailUnauthorized(c, "认证 Token 为空")
			c.Abort()
			return
		}

		// 4. 解析并验证 Token
		claims, err := jwtutil.ParseToken(tokenString, cfg)
		if err != nil {
			logger.Warnw("Failed to parse or validate token", "token", tokenString, "error", err)
			response.FailUnauthorized(c, fmt.Sprintf("认证 Token 无效或已过期: %v", err))
			c.Abort()
			return
		}

		// 5. Token 验证通过，将用户信息存储到 Gin Context 中
		// 后续的 Handler 可以通过 c.Get("userID") 和 c.Get("username") 获取这些信息。
		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		logger.Debugw("JWT authentication successful", UserIDKey, claims.UserID, UsernameKey, claims.Username)

		// 6. 调用请求处理链中的下一个 Handler
		c.Next()
	}
}

// CasbinMiddleware 是一个 Gin 中间件，用于基于 Casbin Enforcer 进行授权检查。
// 它从 Gin Context 中获取当前认证用户的用户名 (由 JWTAuthMiddleware 设置)，
// 以及请求的路径 (Object) 和方法 (Action)。
// 然后使用 Casbin Enforcer 检查用户是否有权限访问该资源和执行该操作。
//
// 参数:
//
//	enforcer: 配置好的 *casbin.Enforcer 实例。
//	logger: Zap SugaredLogger 实例，用于记录授权过程中的信息和错误。
//	cfg: Viper 配置实例 (当前未使用，但保留以备将来扩展，例如从配置中读取匿名用户的角色)。
//
// 返回:
//
//	gin.HandlerFunc: Gin 处理函数，可注册为中间件。
func CasbinMiddleware(enforcer *casbin.Enforcer, logger *zap.SugaredLogger, cfg *viper.Viper) gin.HandlerFunc {
	if enforcer == nil {
		panic("CasbinMiddleware: Casbin Enforcer instance is nil")
	}
	if logger == nil {
		panic("CasbinMiddleware: Zap SugaredLogger instance is nil")
	}
	// cfg 可以为 nil，如果当前中间件不直接使用它。

	return func(c *gin.Context) {
		// 1. 获取请求主体 (Subject)
		// 假设 JWTAuthMiddleware 已将认证用户的用户名存储在 Gin Context 的 UsernameKey 中。
		// 如果没有找到用户名，或者用户未认证，可以将其视为 "anonymous" 角色或直接拒绝。
		var subject string
		usernameAny, userExists := c.Get(UsernameKey) // UsernameKey 来自之前的 JWT 中间件
		if userExists {
			username, ok := usernameAny.(string)
			if ok && username != "" {
				subject = username
			} else {
				// 用户名存在但类型不正确或为空字符串，这可能表示认证流程有问题。
				logger.Warnw("CasbinMiddleware: Username in context is invalid", "username_from_context", usernameAny)
				response.FailUnauthorized(c, "用户认证信息无效 (用户名格式错误)")
				c.Abort()
				return
			}
		} else {
			// 如果上下文中没有用户名，则认为用户是匿名的。
			// 可以根据业务需求定义匿名用户的角色，例如 "role_anonymous"。
			// 或者，如果所有受保护的路由都要求认证用户，则此处可以直接返回 401。
			// 为了简单起见，我们先假设所有需要 Casbin 检查的路由都应有已认证用户。
			// 如果需要支持匿名用户的特定权限，可以在 Casbin 策略中定义 "role_anonymous"。
			// 此处可以设置 subject = "role_anonymous" 或直接拒绝。
			// 根据任务要求，我们先假设如果JWT中间件通过了，那么用户名就存在。
			// 如果JWT中间件未运行或失败，则此中间件不应被调用或JWT中间件已中止请求。
			// 但为了健壮性，添加检查。
			logger.Info("CasbinMiddleware: No username in context, treating as unauthenticated for this middleware (JWT middleware should handle this first)")
			response.FailUnauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		// 2. 获取请求资源 (Object) - 通常是 URL 路径
		obj := c.Request.URL.Path
		// (可选) 根据需要对 obj 进行转换，例如移除查询参数或将其映射到更通用的资源标识符。

		// 3. 获取请求操作 (Action) - 通常是 HTTP 方法
		act := c.Request.Method

		logger.Debugw("CasbinMiddleware: Checking permission", "subject", subject, "object", obj, "action", act)

		// 4. 使用 Casbin Enforcer 执行权限检查
		allowed, err := enforcer.Enforce(subject, obj, act)
		if err != nil {
			// 如果 Enforce 方法本身发生错误 (例如，策略存储问题)，则记录错误并返回服务器内部错误。
			logger.Errorw("Casbin Enforcer.Enforce failed", "subject", subject, "object", obj, "action", act, "error", err)
			response.FailInternalError(c, "权限检查时发生内部错误")
			c.Abort()
			return
		}

		// 5. 根据权限检查结果处理请求
		if allowed {
			logger.Debugw("CasbinMiddleware: Permission granted", "subject", subject, "object", obj, "action", act)
			c.Next() // 用户有权限，继续处理请求链中的下一个 Handler
		} else {
			logger.Warnw("CasbinMiddleware: Permission denied", "subject", subject, "object", obj, "action", act)
			response.FailForbidden(c, "您没有权限执行此操作") // 用户无权限，中止请求并返回 HTTP 403 Forbidden
			c.Abort()
		}
	}
}
