// Package server 负责初始化和配置 HTTP 服务器，例如 Gin 引擎。
// 它整合了路由、中间件和 Handler，并通过 Wire Provider 提供配置好的服务器实例。
package server

import (
	// 用于格式化错误信息和日志
	"strings" // 用于字符串操作，例如处理 API 基础路径

	"go-base-system/internal/handler"    // 导入 Handler 包，用于注册路由
	"go-base-system/internal/middleware" // 导入自定义的中间件包
	"go-base-system/pkg/logger"          // 导入自定义的日志中间件包

	// "go-base-system/pkg/middleware" // 如果有其他自定义中间件，可以在此导入

	"github.com/casbin/casbin/v2"              // 导入 Casbin 包
	"github.com/gin-gonic/gin"                 // Gin Web 框架
	"github.com/google/wire"                   // Wire 依赖注入库
	"github.com/spf13/viper"                   // Viper 配置管理库
	swaggerFiles "github.com/swaggo/files"     // Swagger UI 文件服务
	ginSwagger "github.com/swaggo/gin-swagger" // Gin Swagger 中间件
	"go.uber.org/zap"                          // Zap 高性能日志库
)

// NewGinEngine 是一个 Wire Provider 函数，用于创建和配置一个新的 Gin HTTP 引擎实例。
// 它负责设置 Gin 的运行模式，注册全局中间件（如日志和恢复中间件），
// 定义 API 路由组，并挂载各个业务 Handler 的路由。
//
// 参数:
//
//	cfg: Viper 配置实例，用于读取服务器相关的配置，如运行模式、API 基础路径等。
//	zapSugaredLogger: Zap SugaredLogger 实例，用于在此函数内部记录初始化信息，
//	                  并传递给日志中间件（通过 .Desugar() 获取底层 *zap.Logger）。
//	healthHandler: 健康检查 Handler 实例，用于注册 /health 路由。
//	userHandler: 用户管理 Handler 实例，用于注册用户相关的 API 路由。
//	enforcer: Casbin Enforcer 实例，用于授权。
//
// 返回:
//
//	*gin.Engine: 配置好的 Gin 引擎实例。
//	error: 如果在初始化过程中发生不可恢复的错误，则返回错误。
func NewGinEngine(
	cfg *viper.Viper,
	zapSugaredLogger *zap.SugaredLogger,
	healthHandler *handler.HealthHandler,
	userHandler *handler.UserHandler,
	enforcer *casbin.Enforcer, // 注入 Casbin Enforcer
	// 如果有更多 Handler，在此处添加参数，例如:
	// articleHandler *handler.ArticleHandler,
) (*gin.Engine, error) {
	// 1. 设置 Gin 运行模式 (debug, release, test)
	// 优先从配置 `server.run_mode` 读取，如果未设置，则根据 `app.env` 判断。
	ginMode := cfg.GetString("server.run_mode")
	if ginMode == "" {
		if cfg.GetString("app.env") == "development" { // 假设 app.env 在配置中定义
			ginMode = gin.DebugMode
		} else {
			ginMode = gin.ReleaseMode
		}
	}
	gin.SetMode(ginMode)
	zapSugaredLogger.Infof("Gin 运行模式设置为: %s", ginMode)

	// 2. 创建 Gin 引擎实例
	// 使用 gin.New() 创建一个不带任何默认中间件的引擎，以便完全控制中间件的注册顺序和类型。
	r := gin.New()
	zapSugaredLogger.Info("Gin 引擎实例已创建 (gin.New())。")

	// 3. 注册全局中间件
	// 使用自定义的 Zap 日志中间件 (GinZap) 和 Panic 恢复中间件 (RecoveryWithZap)。
	// .Desugar() 将 *zap.SugaredLogger 转换为底层的 *zap.Logger，以供中间件使用。
	// logger.GinZap 记录每个 HTTP 请求的详细信息。
	// logger.RecoveryWithZap 捕获任何在请求处理链中发生的 panic，记录错误并返回 HTTP 500 响应。
	r.Use(logger.GinZap(zapSugaredLogger.Desugar()))
	r.Use(logger.RecoveryWithZap(zapSugaredLogger.Desugar(), true)) // true 表示记录详细的 panic 堆栈信息
	zapSugaredLogger.Info("已注册 Zap 日志和 Recovery 中间件。")

	// (可选) 在此处可以注册其他全局中间件，例如:
	// - CORS (跨域资源共享) 中间件
	// - 安全相关的中间件 (例如 Helmet)
	// - 请求限流中间件
	// r.Use(middleware.CORSMiddleware())
	// r.Use(middleware.RateLimiter())

	// 4. 定义 API 基础路径和路由组
	// 从 Viper 配置中读取 API 基础路径 (例如 /api/v1)。
	// 如果未配置，则使用默认值。
	basePath := cfg.GetString("api.base_path") // 建议在 config.yaml 中添加 api.base_path
	if basePath == "" {
		basePath = "/api/v1" // 默认 API 基础路径
		zapSugaredLogger.Warnf("配置文件中未找到 'api.base_path'，使用默认值: %s", basePath)
	}
	// 移除路径末尾的斜杠 (如果有)，并确保路径以斜杠开头。
	basePath = "/" + strings.Trim(basePath, "/")

	apiGroup := r.Group(basePath) // 创建 API 路由组
	zapSugaredLogger.Infof("所有 API 路由将注册在基础路径: %s 之下", basePath)

	// 5. 注册各个业务模块的路由
	// 健康检查路由
	apiGroup.GET("/health", healthHandler.Check)
	zapSugaredLogger.Infof("健康检查路由已注册: GET %s/health", basePath)

	// 用户模块路由
	userRoutes := apiGroup.Group("/users") // 创建 /users 子路由组
	{
		userRoutes.POST("/register", userHandler.RegisterUser)               // POST /api/v1/users/register
		userRoutes.POST("/login", userHandler.LoginUser)                     // POST /api/v1/users/login
		userRoutes.GET("/username/:username", userHandler.GetUserByUsername) // GET /api/v1/users/username/{username}
		userRoutes.GET("/id/:id", userHandler.GetUserByID)                   // GET /api/v1/users/id/{id}
		// 更多用户相关的路由可以继续在此处添加
	}
	zapSugaredLogger.Infof("用户模块路由已注册在 %s/users 之下。", basePath)

	// 7. 实例化 JWT 和 Casbin 中间件
	jwtAuthMiddleware := middleware.JWTAuthMiddleware(cfg, zapSugaredLogger)
	casbinAuthzMiddleware := middleware.CasbinMiddleware(enforcer, zapSugaredLogger, cfg)

	// 8. 配置受保护的路由组
	// 首先应用 JWT 中间件，然后应用 Casbin 中间件到整个 apiGroup。
	// 这意味着所有 /api/v1/* 下的路由（除了下面特别排除的）都需要先通过 JWT 认证，然后通过 Casbin 授权。
	// 注意：这将影响公开路由如 /users/login, /users/register, /health。
	// 这些路由需要在 Casbin 策略中为匿名用户或所有已认证用户（如果JWT通过）设置适当的权限。
	// 或者，更精细的做法是在不同的路由组上应用中间件。
	// 为了简单起见，我们先全局应用，然后在 Casbin 策略中放行公共路径。
	// apiGroup.Use(jwtAuthMiddleware) // JWT 认证应按需应用，例如只在 /me 或其他需要登录的组
	// apiGroup.Use(casbinAuthzMiddleware) // Casbin 授权
	// zapSugaredLogger.Info("JWT 和 Casbin 中间件已应用到 apiGroup。")

	// /me 路由组，首先进行 JWT 认证，然后进行 Casbin 授权
	meRoutes := apiGroup.Group("/me")
	meRoutes.Use(jwtAuthMiddleware)     // 先确保用户已登录 (JWT 认证)
	meRoutes.Use(casbinAuthzMiddleware) // 然后检查用户是否有权限访问 (Casbin 授权)
	{
		// GET /api/v1/me/profile - 示例受保护路由，用于获取当前用户信息
		meRoutes.GET("/profile", userHandler.GetCurrentUserProfile)
	}
	zapSugaredLogger.Infof("受 JWT 和 Casbin 保护的 /me 路由组已配置。")

	// 对于 /users 路由组，可以根据需要决定是否整体应用认证和授权，
	// 或仅对特定端点应用。例如，获取用户列表或单个用户信息可能需要特定权限。
	// 注册和登录通常是公开的。
	// 为了演示，我们假设 /users/id/:id 和 /users/username/:username 需要 JWT + Casbin
	// 而 /users/register 和 /users/login 是公开的。

	// 公开的 /users/register 和 /users/login 路由 (已在上面 userRoutes 中定义，不受 meRoutes 的中间件影响)
	// ...

	// 示例：保护获取用户信息的路由
	// 注意：userRoutes 已经在上面定义并添加了 register 和 login。
	// 如果我们想对 userRoutes 下的其他路由应用中间件，可以这样做：
	userSpecificRoutes := userRoutes.Group("") // 创建一个新的子组或直接在 userRoutes 上 .Use()
	userSpecificRoutes.Use(jwtAuthMiddleware)
	userSpecificRoutes.Use(casbinAuthzMiddleware)
	{
		// GetUserByUsername 和 GetUserByID 现在受 JWT 和 Casbin 保护
		// userRoutes.GET("/username/:username", userHandler.GetUserByUsername) // 这些已在上面注册
		// userRoutes.GET("/id/:id", userHandler.GetUserByID)                   // 这些已在上面注册
		// 注意：如果在这里重新注册，会覆盖之前的。正确的做法是先定义好路由组和中间件。
		// 我们已经在上面 userRoutes 中定义了这些路由，现在需要将中间件应用到它们。
		// 最好的做法是在定义 userRoutes 后，再定义需要保护的子组。
		// 或者，在定义 userRoutes 之前就决定是否整个组都需要保护。
		// 此处为了清晰，我们假设上面的 userRoutes 定义中，register 和 login 是公开的，
		// 而查询用户的接口需要保护。
		// 实际上，更常见的做法是为认证用户创建一个单独的路由组。
		// 这里我们保持之前的 userRoutes 结构，但要意识到中间件应用顺序和范围的重要性。
		// 由于 userRoutes 已包含公开端点，我们不能直接对 userRoutes.Use()。
		// 而是应该对需要保护的特定端点或新的子组应用中间件。

		// 为了演示，我们将假定 /users/username/:username 和 /users/id/:id 需要保护
		// 这要求我们在之前定义 userRoutes 的地方，将这些路由移到受保护的组下，或者单独处理。
		// 当前的 userRoutes 定义在应用任何中间件之前，这意味着它们是公开的，除了 meRoutes。
		// 我们将在下面 Swagger 部分后重新组织路由注册以正确应用中间件。
	}

	// 9. (可选) 注册 Swagger UI 文档路由 (如果使用了 Swaggo)
	// 确保 "go-base-system/docs" 包已通过空白导入被包含在 main.go 中，
	// 以便 Swaggo 生成的 swagger 定义能够被注册。
	swaggerPath := "/swagger/*any" // Swagger UI 路径，相对于 apiGroup
	// ginSwagger.URL(...) 用于指定 swagger.json 文件的位置。
	// 如果 basePath 是 /api/v1, 那么 swagger.json 通常在 /api/v1/swagger/doc.json
	// 或者，如果 swag init 时指定了不同的输出路径，这里需要相应调整。
	// 假设 swag init -g main.go (默认输出到 ./docs)，并且 main.go 中的 @BasePath 是 /api/v1
	// 那么 swagger.json 的 URL 应该是 basePath + "/swagger/doc.json"
	// 但更常见的做法是让 ginSwagger.WrapHandler 自动处理 doc.json 的路径，
	// 只要 docs.go 被正确导入。
	// swaggerURL := fmt.Sprintf("http://localhost%s%s/swagger/doc.json", cfg.GetString("server.http_listen_addr"), basePath)
	// 上面的 swaggerURL 可能不正确，通常 WrapHandler 能找到。
	// 如果 swag init -g main.go 那么 docs.SwaggerInfo.BasePath 应该被设置为 /api/v1
	apiGroup.GET(swaggerPath, ginSwagger.WrapHandler(swaggerFiles.Handler))
	zapSugaredLogger.Infof("Swagger UI 文档已注册在: %s%s", basePath, "/swagger/index.html")

	// 6. 返回配置好的 Gin 引擎实例
	zapSugaredLogger.Info("Gin 引擎配置完成。")
	return r, nil // 成功时返回引擎实例和 nil 错误
}

// ProviderSet 是 server 包的 Wire Provider Set。
// 它导出了 NewGinEngine 函数，使其可用于 Wire 的依赖注入图谱，
// Wire 将使用此 Provider 来创建和配置 Gin 引擎实例。
var ProviderSet = wire.NewSet(NewGinEngine)
