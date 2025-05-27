//go:build wireinject
// +build wireinject

// Package bootstrap 负责使用 Wire 进行依赖注入，初始化和引导应用的核心组件。
package bootstrap

import (
	"go-base-system/internal/authz"      // 导入 authz 包 (Casbin)
	"go-base-system/internal/conf"       // 导入配置包
	"go-base-system/internal/db"         // 导入数据库包
	"go-base-system/internal/handler"    // 导入 handler 包
	"go-base-system/internal/repository" // 导入 repository 包
	"go-base-system/internal/server"     // 导入 server 包
	"go-base-system/internal/service"    // 导入 service 包
	"go-base-system/pkg/logger"          // 导入日志包

	"github.com/casbin/casbin/v2" // 导入 Casbin 包
	"github.com/gin-gonic/gin"    // 导入 Gin 包
	"github.com/google/wire"      // 导入 Wire 包
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm" // 导入 GORM 包
)

// App 结构体聚合了应用的核心组件实例。
// 这些实例将由 Wire 自动构建和注入。
type App struct {
	Config   *viper.Viper              // Viper 配置实例
	Logger   *zap.SugaredLogger        // Zap SugaredLogger 实例
	DB       *gorm.DB                  // GORM 数据库连接实例
	UserRepo repository.UserRepository // 用户数据仓库接口实例
	UserSvc  service.UserService       // 用户服务接口实例
	Engine   *gin.Engine               // Gin HTTP 引擎实例
	Enforcer *casbin.Enforcer          // Casbin Enforcer 实例
	// Cleanup func() // Cleanup 函数由 Injector 返回
}

// NewApp 是 App 结构体的构造函数 Provider。
// Wire 会调用此函数来创建 App 实例，并自动注入其依赖项。
func NewApp(
	cfg *viper.Viper,
	zapLogger *zap.SugaredLogger,
	gormDB *gorm.DB,
	userRepo repository.UserRepository,
	userSvc service.UserService,
	engine *gin.Engine, // 注入 Gin 引擎
	enforcer *casbin.Enforcer, // 注入 Casbin Enforcer
) *App {
	return &App{
		Config:   cfg,
		Logger:   zapLogger,
		DB:       gormDB,
		UserRepo: userRepo,
		UserSvc:  userSvc,
		Engine:   engine,   // 赋值 Gin 引擎
		Enforcer: enforcer, // 赋值 Casbin Enforcer
	}
}

// InitializeApp 是我们主要的 Wire injector。
// 它接收 ConfigOptions (包含配置文件路径)，初始化所有核心组件，并返回 App 实例和总的清理函数。
func InitializeApp(cfgOpts conf.ConfigOptions) (*App, func(), error) {
	// wire.Build 指示 Wire 如何构建和连接各个 Provider。
	// Provider 的顺序通常不重要，Wire 会自动解析依赖关系。
	wire.Build(
		// 基础 Providers
		conf.ProviderSet,   // 提供 *viper.Viper
		logger.ProviderSet, // 提供 *zap.SugaredLogger 和清理函数
		db.ProviderSet,     // 提供 *gorm.DB 和清理函数

		// 认证与授权 Providers
		authz.ProviderSet, // 提供 *casbin.Enforcer

		// Repository Providers
		repository.ProviderSet, // 提供 UserRepository

		// Service Providers (现在 service.ProviderSet 内部的 NewUserService 会依赖 *casbin.Enforcer)
		service.ProviderSet, // 提供 UserService

		// Handler Providers
		handler.ProviderSet, // 提供所有 Handler (UserHandler, HealthHandler)

		// Server/Engine Providers
		server.ProviderSet, // 提供 *gin.Engine (现在 server.NewGinEngine 会依赖 *casbin.Enforcer)

		// Top-level App Provider
		NewApp, // App 构造函数，依赖上面提供的实例
	)
	// 这些 nil 返回值会被 Wire 生成的代码替换。
	// Wire 在生成 wire_gen.go 文件时，会用实际的实现替换这部分。
	// 生成的实现将正确处理从 ProvideViper 和 ProvideZapLogger 返回的错误，
	// 并聚合它们的清理函数。
	return nil, nil, nil
}
