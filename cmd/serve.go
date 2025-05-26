/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"go-base-system/internal/handler" // 导入 handler 包
	"go-base-system/pkg/logger"       // 导入日志包
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin" // 导入 Gin
	"github.com/spf13/cobra"
	"github.com/spf13/viper" // 导入 Viper 包
	"go.uber.org/zap"        // 导入 zap，供 ZapGinLogger 使用

	// Swaggo Imports
	// _ "go-base-system/docs" // 这个导入将在 main.go 中处理
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"go-base-system/internal/model" // 导入 User 模型
	"go-base-system/pkg/db"         // 导入数据库初始化包
	"gorm.io/gorm"                  // 导入 gorm
)

// 全局 GORM DB 实例，以便其他包（如 repository）可以访问。
// 注意：全局变量通常不推荐，更好的做法是使用依赖注入（例如 Wire）。
// 这里为了简化示例，暂时使用全局变量。
var gormDB *gorm.DB

// ZapGinLogger 是一个 Gin 中间件，用于使用 Zap 记录请求日志。
// 这是一个可选的自定义中间件示例，如果不想用 Gin 默认的 logger。
func ZapGinLogger(zapLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 请求处理完毕后记录日志
		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			// 如果在处理请求过程中发生错误，记录错误日志
			for _, e := range c.Errors.Errors() {
				zapLogger.Error(e,
					zap.Int("status", c.Writer.Status()),
					zap.String("method", c.Request.Method),
					zap.String("path", path),
					zap.String("query", query),
					zap.String("ip", c.ClientIP()),
					zap.String("user-agent", c.Request.UserAgent()),
					zap.Duration("latency", latency),
				)
			}
		} else {
			// 记录成功的请求
			zapLogger.Info(path,
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.Duration("latency", latency),
			)
		}
	}
}

// serveCmd represents the serve command
// serveCmd 代表 'serve' 子命令，用于启动应用服务。
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动应用服务",
	Long: `启动应用服务。该服务将监听指定的端口并处理传入的请求。
可以通过配置文件或环境变量来配置服务的行为，例如端口号、数据库连接等。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 在命令执行时，首先使用 logger 记录一条信息。
		logger.Log.Info("尝试启动 'serve' 命令...")

		// 初始化数据库连接
		var err error // 在函数级别声明 err
		gormDB, err = db.NewGormDB()
		if err != nil {
			logger.Log.Fatalf("初始化数据库连接失败: %v", err)
			return // 确保在错误发生时退出
		}
		logger.Log.Info("数据库连接初始化成功。")

		// 自动迁移数据库表结构
		// AutoMigrate 会创建表、缺失的外键、约束、列和索引。
		// 它只会创建缺失的内容，不会删除或修改已有的列和索引。
		logger.Log.Info("开始自动迁移数据库表结构...")
		err = gormDB.AutoMigrate(&model.User{}) // 传入 User 模型的指针
		if err != nil {
			logger.Log.Fatalf("数据库自动迁移失败: %v", err)
			return // 确保在错误发生时退出
		}
		logger.Log.Info("数据库表结构自动迁移成功 (User 表)。")

		// 设置 Gin 运行模式
		// Gin 有三种模式: debug, release, test
		// release 模式会关闭 Gin 的一些调试特性，例如详细的请求日志。
		// 可以通过环境变量 GIN_MODE 来设置，或者在代码中设置。
		// 通常在生产环境中设置为 release 模式。
		appEnv := viper.GetString("APP_ENV") // 假设 APP_ENV 在配置中定义，或者通过环境变量传入
		if appEnv == "production" {
			gin.SetMode(gin.ReleaseMode)
			logger.Log.Info("Gin 运行模式设置为: release")
		} else {
			gin.SetMode(gin.DebugMode)
			logger.Log.Info("Gin 运行模式设置为: debug")
		}

		// 初始化 Gin 引擎
		// gin.Default() 返回一个默认的 Gin 引擎，它包含了 Logger 和 Recovery 中间件。
		// Logger 中间件会记录每个请求的日志。
		// Recovery 中间件会在发生 panic 时恢复，并返回一个 500 错误。
		router := gin.Default()

		// 配置 Gin 的日志输出使用 Zap logger
		// 默认的 Gin logger 输出到 os.Stdout。我们可以将其重定向到 Zap。
		// 创建一个 io.Writer，将 Gin 的日志写入 Zap logger。
		ginLogWriter := io.MultiWriter(os.Stdout) // 也可以同时输出到文件或其他地方
		if l, ok := logger.Log.Desugar().Core().(interface{ Write(p []byte) (n int, err error) }); ok {
			ginLogWriter = l // 如果 logger.Log 的底层 Core 支持 Write 方法，则直接使用它
		} else {
			// 作为备选，可以创建一个适配器，但这通常比较复杂，
			// 对于 Gin，通常让其默认 Logger 中间件工作，或者自定义一个新的中间件。
			// 这里我们保持 Gin 默认 Logger (输出到 Stdout)，Zap 用于应用级日志。
			// 或者，完全禁用 Gin 的默认 logger，并使用自定义中间件集成 Zap。
			// router = gin.New() // 使用 gin.New() 创建不带默认中间件的引擎
			// router.Use(gin.Recovery()) // 手动添加 Recovery 中间件
			// router.Use(ZapGinLogger(logger.Log.Desugar())) // 添加自定义的 Zap 日志中间件 (见下方 ZapGinLogger 示例)
			logger.Log.Info("Gin 将使用其默认的 Logger 中间件。应用日志将通过 Zap 处理。")
		}
		// Gin 的默认 Logger 中间件默认写入 gin.DefaultWriter，即 os.Stdout
		// 如果需要更精细控制，可以像上面那样自定义。

		// 定义 API 的基础路径，从 Viper 配置中读取
		// 这个 basePath 将用于 API 端点和 Swagger 文档路径
		basePath := viper.GetString("swagger.basepath")
		if basePath == "" {
			basePath = "/api/v1" // 默认的基础路径
			logger.Log.Warnf("配置文件中未找到 swagger.basepath, 使用默认值: %s", basePath)
		}
		apiGroup := router.Group(basePath)

		// 注册 /health 路由
		// 请求 GET {basePath}/health 将会由 handler.HealthCheck 函数处理。
		// 例如: /api/v1/health
		apiGroup.GET("/health", handler.HealthCheck)
		logger.Log.Infof("健康检查端点注册在: %s/health", basePath)

		// 配置 Swagger UI 路由
		// API 文档路由，例如 /api/v1/swagger/index.html
		// ginSwagger.WrapHandler 需要 swag.Register() 在之前被调用，
		// swag.Register() 通常由 swag init 生成的 docs.go 中的 init() 函数调用。
		// 确保 docs.go 被正确导入（通常在 main.go 或 cmd/root.go）。
		// import _ "go-base-system/docs" // 确保 docs 包被导入以执行其 init 函数
		// docsPath 变量已在上面定义并从配置中获取
		swaggerPath := basePath + "/swagger" // 例如 /api/v1/swagger
		router.GET(swaggerPath+"/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		logger.Log.Infof("Swagger UI 文档地址: http://localhost%s%s/index.html", viper.GetString("server.port"), swaggerPath)


		// 从 Viper 获取服务端口配置。
		serverPort := viper.GetString("server.port")
		if serverPort == "" {
			logger.Log.Warnf("在配置中未找到 'server.port'，将使用默认端口 %s", "8080")
			serverPort = "8080" // 设置一个默认端口
		}

		// 启动 Gin 服务器
		listenAddr := ":" + serverPort
		logger.Log.Infof("Gin 服务器准备在地址 %s 上启动...", listenAddr)

		// 异步启动服务器，以便可以进行优雅关闭等操作（如果需要）
		// router.Run() 是一个阻塞操作。
		if err := router.Run(listenAddr); err != nil {
			logger.Log.Fatalf("Gin 服务器启动失败: %v", err)
		}

		// 这行通常不会执行到，因为 router.Run() 是阻塞的，除非发生错误。
		logger.Log.Info("'serve' 命令执行完毕。")
	},
}

func init() {
	// 将 serveCmd 添加到 rootCmd，这样 Cobra 应用就能识别并执行它。
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings for the serve command.
	// 在这里为 'serve' 命令定义标志和配置设置。

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")
	// Persistent Flags (持久标志) 在此命令及其所有子命令中都可用。
	// 例如，可以为 serve 命令添加一个持久标志来指定数据目录：
	// serveCmd.PersistentFlags().String("data-dir", "/var/lib/myapp", "数据存储目录")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	// Local Flags (本地标志) 仅在此命令直接调用时运行。
	// 例如，可以为 serve 命令添加一个本地标志来启用调试模式：
	// serveCmd.Flags().BoolP("debug", "d", false, "启用调试模式 (仅用于 serve 命令)")

	// 还可以将 Viper 绑定到命令的标志。
	// 例如，如果 serveCmd 有一个名为 "port" 的标志：
	// serveCmd.Flags().IntP("port", "p", 8080, "服务监听的端口")
	// viper.BindPFlag("server.port", serveCmd.Flags().Lookup("port"))
	// 这样，如果用户通过 --port 命令行参数指定了端口，Viper 就能获取到这个值。
	// 注意：如果同时在配置文件、环境变量和命令行标志中设置了同一个值，Viper 有一套优先级规则（通常命令行 > 环境变量 > 配置文件）。
}

// 注释掉之前的 ZapGinLogger 和相关导入信息，因为它们已经被移到文件顶部。
