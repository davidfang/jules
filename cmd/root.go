/*
Copyright © 2025 Your Name <your.email@example.com>
*/
package cmd

import (
	"fmt"
	// "os" // os.Exit(1) 可能会在 PersistentPreRunE 中使用
	// "strings" // strings.NewReplacer 可能会在 initConfig 中使用 (如果保留 initConfig)

	"github.com/spf13/cobra"
	// "github.com/spf13/viper" // Viper 将由 Wire 注入
	"go-base-system/internal/bootstrap" // 导入 bootstrap 包
	"go-base-system/internal/conf"      // 导入 conf 包，用于 ConfigOptions
	// "go-base-system/pkg/logger" // Logger 将从 AppInstance 获取
	// "go.uber.org/zap" // Zap 将从 AppInstance 获取
)

// cfgFile 用于存储通过命令行标志指定的配置文件路径。
var cfgFile string

// AppInstance 是由 Wire 初始化并包含所有核心应用组件的实例。
// 它作为包级全局变量，以便其他命令可以访问 Logger, Config 等。
// 注意：这是一种简化的依赖传递方式。在更复杂的应用中，
// 可能会选择通过 Context 或其他方式将依赖项传递给子命令。
var AppInstance *bootstrap.App

// AppCleanupFunc 用于存储从 InitializeApp 返回的清理函数。
var AppCleanupFunc func()

// rootCmd 代表应用的基础命令，在不带任何子命令的情况下被调用。
var rootCmd = &cobra.Command{
	Use:   "go-base-system-v2",
	Short: "Go 基础系统 v2 是一个现代化的 Go 应用脚手架。",
	Long: `Go 基础系统 v2 (go-base-system-v2) 旨在提供一个
健壮且可扩展的起点，用于快速开发高质量的 Go 应用。
它集成了 Cobra, Viper, Zap, Wire 等流行的 Go 库。`,
	// PersistentPreRunE 在所有子命令的 RunE (或 Run) 之前执行。
	// 我们在这里使用它来初始化应用的核心组件 (Viper, Logger) 通过 Wire。
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. 创建 ConfigOptions，将命令行标志传入的 cfgFile 路径包含进去。
		configOptions := conf.ConfigOptions{FilePath: cfgFile}

		// 2. 调用 Wire 生成的 InitializeApp injector。
		var err error
		AppInstance, AppCleanupFunc, err = bootstrap.InitializeApp(configOptions)
		if err != nil {
			// 如果初始化失败 (例如，指定的配置文件无法读取且格式错误)，
			// 则返回错误，Cobra 会打印错误并退出。
			return fmt.Errorf("应用初始化失败: %w", err)
		}

		// 初始化成功后，可以立即使用 Logger。
		AppInstance.Logger.Info("应用核心组件初始化成功 (通过 Wire)。")
		if AppInstance.Config.ConfigFileUsed() != "" {
			AppInstance.Logger.Infof("成功加载配置文件: %s", AppInstance.Config.ConfigFileUsed())
		} else {
			AppInstance.Logger.Info("未找到或未使用配置文件，将依赖环境变量或默认值。")
		}
		// AppInstance.Logger.Debugw("所有已加载的配置项", "settings", AppInstance.Config.AllSettings())

		return nil // 初始化成功
	},
}

// Execute 函数执行 rootCmd 命令。
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// 移除 cobra.OnInitialize(initConfig) 调用，因为初始化逻辑移至 PersistentPreRunE。

	// 定义 --config 持久标志。
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "指定配置文件路径 (例如: ./config/config.yaml)")

	// 注册 AppCleanupFunc，使其在 Cobra 命令执行完毕后被调用。
	// OnFinalize 确保即使命令执行出错，清理函数也会被调用。
	cobra.OnFinalize(func() {
		if AppCleanupFunc != nil {
			if AppInstance != nil && AppInstance.Logger != nil { // 确保 Logger 存在
				AppInstance.Logger.Info("执行应用清理操作...")
			} else {
				fmt.Println("执行应用清理操作... (Logger 未初始化)")
			}
			AppCleanupFunc()
		}
	})
}

// initConfig 函数不再需要，其功能已由 internal/conf/ProvideViper 和
// rootCmd.PersistentPreRunE 中的 bootstrap.InitializeApp 调用处理。
// 旧的 initConfig 函数内容可以删除或注释掉。
/*
func initConfig() {
	// ... (旧的 Viper 和 Logger 初始化代码) ...
}
*/
