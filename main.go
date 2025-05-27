/*
Copyright © 2025 Your Name <your.email@example.com>
*/

// @title           Go Base System API V2
// @version         1.0
// @description     这是一个使用 Golang (Gin, GORM, Wire) 构建的基础系统 API 服务第二版。
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    https://www.example.com/support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1 // **确保与 NewGinEngine 中注册的路由组路径一致**

// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Example: "Bearer {token}"

package main

import (
	"go-base-system/cmd"
	_ "go-base-system/docs" // 导入 Swaggo 生成的 docs 包
	// "fmt"
	// "os"
)

func main() {
	// 应用的核心初始化 (Viper, Logger) 现在通过 cmd/root.go 中的
	// rootCmd.PersistentPreRunE -> bootstrap.InitializeApp() 完成。
	// cmd.Execute() 会触发这个流程。

	// cmd.Execute() 内部已使用 cobra.CheckErr 处理错误，
	// 该函数会在发生错误时打印错误并以状态码 1 退出。
	// 因此，这里通常不需要再对 cmd.Execute() 的返回值进行错误处理。
	cmd.Execute()

	// 清理函数 (AppCleanupFunc) 已通过 cobra.OnFinalize 在 cmd/root.go 中注册，
	// 会在 cmd.Execute() 即将返回或程序因 cobra.CheckErr 退出前被调用。
	// 无需在此处显式调用清理。

	// 如果需要在 cmd.Execute() 之后执行特定逻辑（例如，如果 Execute 不会阻塞或出错），
	// 可以在这里添加。但通常，Cobra 应用的生命周期由 Execute() 管理。
	// 如果 cmd.Execute() 内部启动了常驻服务（如 HTTP 服务器），
	// 那么程序可能不会执行到 Execute() 之后的代码，除非服务关闭。
}
