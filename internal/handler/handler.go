// Package handler 提供了应用中所有 HTTP Handler 的 Wire Provider Set。
// 这个文件作为一个聚合点，方便在 Injector 中统一导入所有 Handler相关的 Provider。
package handler

import (
	"github.com/google/wire" // 导入 Wire 包
)

// ProviderSet 将所有 Handler 的构造函数 Provider 聚合在一起。
// Wire 在构建依赖图时会使用这个集合来提供 Handler 实例。
// 如果未来添加了新的 Handler (例如 ProductHandler, OrderHandler 等)，
// 它们的构造函数 Provider (例如 NewProductHandler, NewOrderHandler) 也应在此处添加。
var ProviderSet = wire.NewSet(
	NewHealthHandler, // 健康检查 Handler 的 Provider
	NewUserHandler,   // 用户相关 Handler 的 Provider
	// 未来可以添加更多 Handler Provider，例如:
	// NewArticleHandler,
	// NewOrderHandler,
)
