// Package handler 包含了处理 HTTP 请求的 Handler (控制器)。
// Handler 层负责解析请求、调用 Service 层处理业务逻辑，并使用 pkg/response 返回响应。
package handler

import (
	"net/http" // 导入 net/http 包，用于 HTTP 状态码常量。

	"go-base-system-v2/pkg/response" // 导入自定义的响应包。

	"github.com/gin-gonic/gin" // 导入 Gin 框架包。
	"go.uber.org/zap"          // 导入 Zap 日志库。
	// "github.com/google/wire" // Wire ProviderSet 将在 handler.go 中统一定义
)

// HealthHandler 结构体封装了健康检查相关的依赖。
// 当前它只依赖于 Zap SugaredLogger 进行日志记录。
type HealthHandler struct {
	logger *zap.SugaredLogger // Zap SugaredLogger 实例，用于记录日志。
}

// NewHealthHandler 是 HealthHandler 的构造函数 Provider。
// 此函数用于 Wire 进行依赖注入，创建一个新的 HealthHandler 实例。
// 参数:
//   logger: Zap SugaredLogger 实例。如果为 nil，函数将 panic。
// 返回:
//   *HealthHandler: 指向 HealthHandler 实例的指针。
func NewHealthHandler(logger *zap.SugaredLogger) *HealthHandler {
	if logger == nil {
		panic("NewHealthHandler: Zap SugaredLogger instance is nil")
	}
	return &HealthHandler{logger: logger}
}

// Check godoc
// @Summary      服务健康检查 (Service Health Check)
// @Description  检查 API 服务是否正常运行并可达。
// @Tags         健康检查 (Health Check)
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.ResponseData{data=map[string]string}  "成功" // 更新 Success 注解
// @Router       /health [get] // 注意：此路由路径是相对于 API Group 的基础路径。例如 /api/v1/health
// Check 方法是 HealthHandler 的一个处理函数，用于处理健康检查的 HTTP GET 请求。
// 它使用 pkg/response 包中的 Success 函数返回一个标准的 JSON 响应，
// 表明服务状态为 "UP"。
func (h *HealthHandler) Check(c *gin.Context) {
	h.logger.Debug("执行健康检查 API 端点 (Health Check endpoint executed)") // 记录调试信息

	// 使用 response.SuccessOK 发送一个 HTTP 200 OK 响应。
	// gin.H 是 map[string]interface{} 的一个快捷方式，常用于构建 JSON 对象。
	response.SuccessOK(c, gin.H{"status": "UP", "message": "Service is running healthy."})
}
