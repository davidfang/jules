/*
Copyright © 2025 Your Name <your.email@example.com>
*/
package cmd

import (
	"context" // 导入 context 包
	// 导入 errors 包
	"fmt" // 导入 fmt 用于错误处理

	"go-base-system/internal/dto"   // 导入 DTO 包
	"go-base-system/internal/model" // 导入 User 模型
	"go-base-system/pkg/translator" // 导入翻译器包

	"github.com/spf13/cobra"
	// "go-base-system/internal/repository" // repository.ErrNotFound 已通过 errors.Is 检查，直接导入 errors 即可
	// Viper 和 Logger 实例将通过 cmd.AppInstance 访问
)

// serverCmd 代表 'server' 子命令，用于启动应用的主要服务（例如 HTTP 服务器）。
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动应用服务",
	Long: `启动应用服务。根据配置文件中的设置，此命令可以启动一个 HTTP 服务器，
监听指定的端口，并开始处理传入的请求。

例如:
  go-base-system-v2 server -c ./config/config.yaml

服务启动后，相关的运行时信息和错误将通过配置的日志系统输出。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 检查 AppInstance 是否已通过 rootCmd 的 PersistentPreRunE 初始化。
		if AppInstance == nil || AppInstance.Logger == nil || AppInstance.Config == nil || AppInstance.DB == nil || AppInstance.UserSvc == nil || AppInstance.Engine == nil {
			return fmt.Errorf("应用核心组件未完全初始化。请检查 rootCmd 的 PersistentPreRunE 设置和 Wire 配置")
		}

		logger := AppInstance.Logger
		config := AppInstance.Config
		db := AppInstance.DB
		userSvc := AppInstance.UserSvc
		engine := AppInstance.Engine

		logger.Info("服务器子命令 'server' 被调用")
		logger.Debugf("配置文件路径 (来自 --config 标志，如有提供): %s", cfgFile)
		logger.Debugf("实际使用的配置文件 (由 Viper 决定): %s", config.ConfigFileUsed())

		serverPort := config.GetString("server.port")
		if serverPort == "" {
			logger.Warn("在配置中未找到 'server.port'，将使用默认端口 8080 (示例)")
			serverPort = "8080"
		}
		logger.Infof("计划在端口 %s 上启动服务 (来自配置)", serverPort)

		logger.Info("开始自动迁移数据库表结构 (User 表)...")
		if err := db.AutoMigrate(&model.User{}); err != nil {
			logger.Errorf("数据库自动迁移失败: %v", err)
			return fmt.Errorf("数据库自动迁移失败: %w", err)
		}
		logger.Info("数据库表结构自动迁移成功 (User 表)。")

		logger.Info("正在测试数据库连接 (执行 SELECT 1)...")
		if err := db.Exec("SELECT 1").Error; err != nil {
			logger.Errorf("数据库连接测试失败: %v", err)
			return fmt.Errorf("数据库连接测试失败: %w", err)
		}
		logger.Info("数据库连接测试成功。")

		// ---- UserService 初步测试 (可选，生产环境应移除或通过标志控制) ----
		if config.GetString("app.env") == "development" { // 例如，只在开发环境运行测试
			logger.Info("开始 UserService 初步测试 (仅开发环境)...")
			ctx := context.Background()
			testEmail := "servicetest@example.com"
			registerReq := &dto.UserRegisterReq{
				Username: "servicetestuser",
				Email:    testEmail,
				Password: "password123",
			}
			// 清理可能存在的旧测试用户 (简单处理，实际测试可能需要更完善的 setup/teardown)
			_ = db.Where("username = ? OR email = ?", registerReq.Username, registerReq.Email).Delete(&model.User{}).Error

			registeredUser, err := userSvc.RegisterUser(ctx, registerReq)
			if err != nil {
				logger.Errorf("通过 Service 注册用户 '%s' 失败: %v", registerReq.Username, err)
			} else {
				logger.Infof("通过 Service 注册用户 '%s' 成功, ID: %d, Email: %s", registeredUser.Username, registeredUser.ID, *registeredUser.Email)
				foundUser, err := userSvc.GetUserByUsername(ctx, registerReq.Username)
				if err != nil {
					logger.Errorf("通过 Service 查询用户 '%s' 失败: %v", registerReq.Username, err)
				} else {
					logger.Infof("通过 Service 查询用户 '%s' 成功: Username=%s, Email=%s, ID=%d", foundUser.Username, *foundUser.Email, foundUser.ID)
				}
			}
			logger.Info("UserService 初步测试完成。")
		}
		// ---- UserService 初步测试 (结束) ----

		// 初始化校验翻译器
		logger.Info("正在初始化校验翻译器...")
		if err := translator.InitTranslator(); err != nil {
			logger.Fatalf("初始化校验翻译器失败: %v", err)
			return fmt.Errorf("初始化校验翻译器失败: %w", err)
		}
		logger.Info("校验翻译器初始化成功。")

		listenAddr := ":" + serverPort
		logger.Infof("准备在地址 %s 上启动 HTTP Gin 服务器...", listenAddr)
		if err := engine.Run(listenAddr); err != nil {
			logger.Fatalf("Gin 服务器启动失败: %v", err)
			return fmt.Errorf("Gin 服务器启动失败: %w", err)
		}

		logger.Info("Gin 服务器已关闭。")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
