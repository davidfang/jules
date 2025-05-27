// Package authz (Authorization) 负责应用的授权逻辑，主要使用 Casbin。
// 它提供了 Casbin Enforcer 的初始化 Provider，并可能包含其他授权相关的工具函数。
package authz

import (
	"fmt"
	"time" // 用于策略自动加载的时间间隔

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3" // GORM Adapter for Casbin
	"github.com/google/wire"                        // Wire 依赖注入库
	"github.com/spf13/viper"                        // Viper 配置管理库
	"go.uber.org/zap"                               // Zap 高性能日志库
	"gorm.io/gorm"                                  // GORM 数据库操作库
)

// ProvideCasbinEnforcer 是一个 Wire Provider 函数，用于创建和配置 Casbin Enforcer 实例。
// Enforcer 是 Casbin 的核心组件，用于执行权限检查。
//
// 参数:
//   cfg: Viper 配置实例，用于读取 Casbin 模型文件路径和策略自动加载配置。
//   db: GORM 数据库连接实例，用于 Casbin GORM Adapter 持久化策略。
//   logger: Zap SugaredLogger 实例，用于记录 Enforcer 初始化和策略加载过程中的信息。
//
// 返回:
//   *casbin.Enforcer: 配置好的 Casbin Enforcer 实例。
//   error: 如果在初始化过程中发生错误 (例如模型文件未找到、数据库适配器创建失败、策略加载失败等)，则返回错误。
func ProvideCasbinEnforcer(cfg *viper.Viper, db *gorm.DB, logger *zap.SugaredLogger) (*casbin.Enforcer, error) {
	modelPath := cfg.GetString("casbin.model_path")
	if modelPath == "" {
		return nil, fmt.Errorf("casbin 模型文件路径 (casbin.model_path) 未在配置中设置")
	}
	logger.Infof("Casbin: 正在加载模型文件从路径: %s", modelPath)

	// 1. 创建 GORM Adapter
	// GORM Adapter 用于将 Casbin 的策略规则持久化到数据库中。
	// 它会自动创建名为 "casbin_rule" 的表来存储策略。
	// 第二个参数 (布尔值) 表示是否在数据库中创建表（如果尚不存在）。通常设为 false，因为我们希望 GORM AutoMigrate 来管理表创建。
	// 但 gorm-adapter 的 NewAdapterByDB 行为是如果表不存在会自动创建，所以这里参数影响不大。
	// 第三个参数是表名，如果为空字符串，则默认为 "casbin_rules"。
	adapter, err := gormadapter.NewAdapterByDB(db) // 使用默认表名 "casbin_rules"
	if err != nil {
		logger.Errorw("创建 Casbin GORM Adapter 失败", "error", err)
		return nil, fmt.Errorf("创建 Casbin GORM Adapter 失败: %w", err)
	}
	logger.Info("Casbin GORM Adapter 创建成功。")

	// 2. 创建 Casbin Enforcer
	// NewEnforcer 从模型文件和策略适配器创建一个 Enforcer 实例。
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		logger.Errorw("创建 Casbin Enforcer 失败", "model_path", modelPath, "error", err)
		return nil, fmt.Errorf("创建 Casbin Enforcer 失败: %w", err)
	}
	logger.Info("Casbin Enforcer 创建成功。")

	// 3. 从数据库加载策略规则
	// LoadPolicy() 方法将存储在数据库中的所有策略规则加载到 Enforcer 中。
	// 如果数据库中没有策略规则，或者这是首次启动，此操作不会报错。
	if err := enforcer.LoadPolicy(); err != nil {
		logger.Errorw("从数据库加载 Casbin 策略失败", "error", err)
		return nil, fmt.Errorf("加载 Casbin 策略失败: %w", err)
	}
	logger.Info("Casbin 策略已从数据库加载。")

	// 4. 添加默认策略 (如果不存在)
	// 为确保应用有基础的权限设置，可以在此处添加一些默认策略。
	// 在添加前检查策略是否已存在，以避免重复。
	defaultPolicies := [][]string{
		// role_admin 可以对 /api/v1/users/ 下的所有资源执行任何 HTTP 方法
		{"role_admin", "/api/v1/users/*", "(GET)|(POST)|(PUT)|(DELETE)"},
		// role_admin 可以 GET /api/v1/me/profile
		{"role_admin", "/api/v1/me/profile", "GET"},
		// role_user 可以 GET /api/v1/me/profile
		{"role_user", "/api/v1/me/profile", "GET"},
		// 示例：允许所有用户访问健康检查接口
		{"role_anonymous", "/api/v1/health", "GET"}, // 假设存在匿名角色或所有人都可访问
		{"role_user", "/api/v1/health", "GET"},
		{"role_admin", "/api/v1/health", "GET"},
	}

	for _, policy := range defaultPolicies {
		// HasPolicy 检查策略是否已存在
		// Casbin v2.83.1 中 HasPolicy 接受 []string 作为参数
		if has, _ := enforcer.HasPolicy(policy[0], policy[1], policy[2]); !has {
			// AddPolicy 添加一条策略规则
			if _, err := enforcer.AddPolicy(policy[0], policy[1], policy[2]); err != nil {
				logger.Errorw("添加默认 Casbin 策略失败", "policy", policy, "error", err)
				// 根据需求决定是否返回错误，或者只是记录并继续
			} else {
				logger.Infof("成功添加默认 Casbin 策略: %v", policy)
			}
		} else {
			logger.Debugf("默认 Casbin 策略已存在，跳过添加: %v", policy)
		}
	}
	// (可选) 添加默认的角色分配，例如：
	// if hasRole, _ := enforcer.HasRoleForUser("admin_user_example", "role_admin"); !hasRole {
	//    enforcer.AddRoleForUser("admin_user_example", "role_admin")
	// }

	// 5. （可选）启用策略自动加载
	// 如果配置文件中启用了自动加载策略，则启动一个 goroutine 定期从数据库重新加载策略。
	if cfg.GetBool("casbin.auto_load_policy") {
		intervalMinutes := cfg.GetDuration("casbin.auto_load_policy_interval_minutes")
		if intervalMinutes == 0 {
			intervalMinutes = 1 // 默认为 1 分钟
		}
		go func() {
			ticker := time.NewTicker(intervalMinutes * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				if err := enforcer.LoadPolicy(); err != nil {
					logger.Errorw("自动加载 Casbin 策略失败", "error", err)
				} else {
					logger.Info("Casbin 策略已重新加载。")
				}
			}
		}()
		logger.Infof("Casbin 策略自动加载已启用，间隔: %v 分钟。", intervalMinutes)
	}

	logger.Info("Casbin Enforcer 初始化并配置完成。")
	return enforcer, nil
}

// ProviderSet 是 authz 包的 Wire Provider Set。
// 它导出了 ProvideCasbinEnforcer 函数，使其可用于 Wire 的依赖注入图谱。
var ProviderSet = wire.NewSet(ProvideCasbinEnforcer)
