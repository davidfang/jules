package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/wire" // 导入 Wire 包
	"go.uber.org/zap"        // 导入 Zap SugaredLogger
)

// ProviderSet 是 handler 包中与健康检查相关的 Wire Provider Set。
// 它导出了 NewHealthHandler 函数。
var ProviderSet = wire.NewSet(NewHealthHandler)

// HealthHandler 结构体封装了健康检查相关的依赖，例如 logger。
type HealthHandler struct {
	logger *zap.SugaredLogger // Zap SugaredLogger 实例，用于记录日志
}

// NewHealthHandler 是 HealthHandler 的构造函数。
// 它接收一个 *zap.SugaredLogger 实例作为参数，并返回一个 *HealthHandler。
// 这个构造函数使得依赖注入（DI）更容易实现。
func NewHealthHandler(logger *zap.SugaredLogger) *HealthHandler {
	return &HealthHandler{
		logger: logger,
	}
}

// Check godoc
// @Summary      服务健康检查
// @Description  检查API服务是否正常运行
// @Tags         健康检查
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  // 使用 map[string]interface{} 作为示例
// @Router       /health [get] // 注意：这个路由路径是相对于 API Group 的基础路径的
// Check 方法用于处理健康检查请求。
// 它现在是 HealthHandler 结构体的一个方法，可以访问 handler 内部的依赖（如 logger）。
func (h *HealthHandler) Check(c *gin.Context) {
	// 使用注入的 logger 记录一条信息。
	h.logger.Debug("执行健康检查 API 端点...")

	// c.JSON 是 Gin 框架提供的一个便捷方法，用于发送 JSON 格式的响应。
	// http.StatusOK 是一个常量，表示 HTTP 状态码 200 (OK)。
	// gin.H 是 map[string]interface{} 的一个快捷方式，常用于构建 JSON 对象。
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"message": "服务运行正常 (v1, DI)", // 消息更新以反映更改
	})
}
