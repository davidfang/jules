// Package logger 提供了 Zap 日志库的初始化和配置功能。
package logger

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin" // 导入 gin
	"github.com/google/wire"   // 导入 Wire 包
	"github.com/spf13/viper"   // 导入 Viper 包
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ProvideZapLogger 根据 Viper 配置初始化 Zap SugaredLogger。
// 它从 Viper 实例中读取日志相关的配置项，构建 zap.Config，
// 然后创建 *zap.Logger 并转换为 *zap.SugaredLogger。
// 返回一个配置好的 *zap.SugaredLogger 实例，一个用于同步日志的清理函数，以及可能发生的错误。
func ProvideZapLogger(cfg *viper.Viper) (*zap.SugaredLogger, func(), error) {
	// 从 Viper 配置中读取日志级别。
	logLevel := cfg.GetString("logger.level")
	if logLevel == "" {
		logLevel = "info" // 默认日志级别
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		// 如果级别字符串无效，则返回错误。
		return nil, nil, fmt.Errorf("无效的日志级别 '%s': %w。请使用 debug, info, warn, error, dpanic, panic, fatal", logLevel, err)
	}

	logFormat := cfg.GetString("logger.format")
	if logFormat == "" {
		logFormat = "console"
	}

	outputPaths := cfg.GetStringSlice("logger.output_paths")
	if len(outputPaths) == 0 {
		outputPaths = []string{"stdout"}
	}
	errorOutputPaths := cfg.GetStringSlice("logger.error_output_paths")
	if len(errorOutputPaths) == 0 {
		errorOutputPaths = []string{"stderr"}
	}

	development := cfg.GetBool("logger.development")
	disableCaller := cfg.GetBool("logger.disable_caller")
	disableStacktrace := cfg.GetBool("logger.disable_stacktrace")

	var encoderConfig zapcore.EncoderConfig
	if logFormat == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.MessageKey = "message"
		encoderConfig.LevelKey = "level"
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.CallerKey = "caller"
		encoderConfig.StacktraceKey = "stacktrace"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       development,
		DisableCaller:     disableCaller,
		DisableStacktrace: disableStacktrace,
		Encoding:          logFormat,
		EncoderConfig:     encoderConfig,
		OutputPaths:       outputPaths,
		ErrorOutputPaths:  errorOutputPaths,
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("Zap logger 构建失败: %w", err)
	}

	sugaredLogger := logger.Sugar()
	cleanup := func() {
		_ = sugaredLogger.Sync()
	}

	sugaredLogger.Debugf("Zap SugaredLogger 初始化成功。级别: %s, 格式: %s", level.String(), logFormat)
	return sugaredLogger, cleanup, nil
}

// ProviderSet 是 logger 包的 Wire Provider Set。
// 它导出了 ProvideZapLogger 函数。
var ProviderSet = wire.NewSet(ProvideZapLogger)

// GinZap 是一个 Gin 中间件，使用 Zap 进行 HTTP 请求日志记录。
// 它接收一个 *zap.Logger 实例（非 SugaredLogger）。
func GinZap(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next() // 处理请求

		latency := time.Since(start)

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() { // gin.Error 类型实现了 error 接口
				logger.Error(e, // 直接记录错误信息
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
			logger.Info(path, // 对于成功请求，路径作为消息
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path), // 路径也作为字段记录
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.Duration("latency", latency),
			)
		}
	}
}

// RecoveryWithZap 返回一个 Gin 中间件，该中间件从任何 panic 中恢复，
// 如果发生 panic，则写入 500 错误，并使用 Zap 记录错误。
// 它接收一个 *zap.Logger 实例（非 SugaredLogger）和是否记录堆栈信息的标志。
func RecoveryWithZap(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Error(c.Request.URL.Path, // 路径作为消息
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					logger.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					logger.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
