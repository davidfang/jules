// Package conf 负责应用配置的加载和管理。
// 它使用 Viper 库来处理配置文件、环境变量等。
package conf

import (
	"fmt"
	"strings" // 用于环境变量键名替换

	"github.com/fsnotify/fsnotify" // 用于监控配置文件变化
	"github.com/google/wire"      // 导入 Wire 包
	"github.com/spf13/viper"
)

// ConfigOptions 包含加载 Viper 配置所需的参数。
// FilePath 是配置文件的可选路径，如果提供，则 Viper 会尝试加载此文件。
type ConfigOptions struct {
	FilePath string // 配置文件路径，通常由命令行参数传入
}

// NewConfigOptions 是 ConfigOptions 的构造函数 Provider。
// 在当前的早期集成阶段，ConfigOptions 实例将直接在 InitializeApp injector 中创建并传递。
// 如果 FilePath 需要从其他依赖项注入，则可以取消注释并使用此 Provider。
// func NewConfigOptions(filePath string) ConfigOptions {
// 	return ConfigOptions{FilePath: filePath}
// }

// ProvideViper 根据提供的 ConfigOptions 初始化并返回一个配置好的 *viper.Viper 实例。
// 它负责设置配置文件路径、名称、类型，并启用环境变量的自动读取和前缀设置。
// 还包括了对配置文件变化的监控功能。
// 如果在读取指定配置文件时发生错误，将返回错误。
func ProvideViper(opts ConfigOptions) (*viper.Viper, error) {
	v := viper.New() // 创建一个新的 Viper 实例

	if opts.FilePath != "" {
		// 如果通过 ConfigOptions 指定了配置文件路径，则使用该路径。
		v.SetConfigFile(opts.FilePath)
		// fmt.Printf("Viper: 使用指定的配置文件: %s\n", opts.FilePath) // 调试日志
	} else {
		// 如果未指定配置文件路径，则按顺序搜索默认路径。
		v.AddConfigPath("./config")                 // 1. 当前工作目录下的 config 子目录
		v.AddConfigPath("$HOME/.go-base-system-v2") // 2. 用户主目录下的 .go-base-system-v2 子目录
		v.SetConfigName("config")                   // 配置文件的名称（不带扩展名）
		v.SetConfigType("yaml")                     // 明确指定配置文件类型为 YAML
		// fmt.Println("Viper: 未指定配置文件路径，将搜索默认路径...") // 调试日志
	}

	// 尝试读取配置文件。
	if err := v.ReadInConfig(); err != nil {
		// 检查错误类型是否为配置文件未找到错误。
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 如果用户明确指定了配置文件路径但文件未找到，则这是一个需要报告的错误。
			if opts.FilePath != "" {
				return nil, fmt.Errorf("指定的配置文件 '%s' 未找到: %w", opts.FilePath, err)
			}
			// 如果未指定配置文件路径，并且在默认路径中也未找到，则忽略此错误。
			// 应用可以继续运行，依赖环境变量或默认值进行配置。
			// fmt.Println("Viper: 未找到配置文件，将依赖环境变量或默认值。") // 调试日志
		} else {
			// 如果是其他类型的错误（例如，配置文件格式错误），则返回错误。
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	} else {
		// 配置文件成功加载。
		// fmt.Printf("Viper: 成功加载配置文件: %s\n", v.ConfigFileUsed()) // 调试日志
	}

	// 从配置文件中读取应用的环境变量前缀。
	// 如果配置文件中未设置，或者配置文件未加载，则使用默认值 "APP"。
	envPrefix := v.GetString("app.env_prefix") // 假设配置文件中有 app.env_prefix
	if envPrefix == "" {
		envPrefix = "APP" // 默认的环境变量前缀
	}
	v.SetEnvPrefix(envPrefix) // 设置 Viper 从环境变量读取时的前缀

	// 启用 Viper 从环境变量中自动读取配置的功能。
	v.AutomaticEnv()

	// 设置环境变量键名替换规则。
	// 例如，配置文件中的 server.port，对应的环境变量是 APP_SERVER_PORT (如果前缀是 APP)。
	// Viper 在查找环境变量时，会将键名中的 "." 替换为 "_"。
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// （可选）启用配置文件监控功能。
	// 当配置文件发生变化时，Viper 可以自动重新加载配置。
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		// 配置文件变化时的回调函数。
		// 注意：此时 logger 可能尚未完全初始化或通过 DI 注入，因此使用 fmt 打印。
		fmt.Printf("通知: 配置文件 '%s' 发生变化 (%s)\n", e.Name, e.Op)
		// 可以在这里添加重新加载受影响组件配置的逻辑，
		// 或者触发应用级别的配置刷新事件。
		// 例如: v.ReadInConfig() // 重新读取配置，但要注意并发安全
	})

	// 调试日志：打印一些关键配置项，确认 Viper 加载正确。
	// 此处应在 logger 初始化后使用 logger 实例。
	// fmt.Printf("Viper: 完成配置加载。Server Port (from config): %s, App Env Prefix: %s\n", v.GetString("server.port"), envPrefix)

	return v, nil
}

// ProviderSet 是 conf 包的 Wire Provider Set。
// 它导出了 ProvideViper 函数，使其可用于 Wire 的依赖注入图谱。
// 如果 NewConfigOptions 被激活，也应包含在这里。
var ProviderSet = wire.NewSet(ProvideViper)
