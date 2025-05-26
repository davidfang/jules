// Package db 负责数据库连接的初始化和管理。
// 它使用 GORM 作为 ORM 工具，并提供了通过 Wire 进行依赖注入的 Provider。
package db

import (
	"context" // 导入 context 包
	"fmt"
	"strings" // 导入 strings 包，用于 GORM Logger 配置
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/google/wire" // 导入 Wire 包
)

// ZapGormLogger 实现 gormlogger.Interface，将 GORM 日志桥接到 Zap。
// 这样 GORM 的内部日志就可以通过我们配置的 Zap logger 来输出和管理。
type ZapGormLogger struct {
	ZapLogger     *zap.Logger         // Zap Logger 实例 (非 SugaredLogger，以支持结构化字段)
	LogLevel      gormlogger.LogLevel // GORM 的日志级别 (Silent, Error, Warn, Info)
	SlowThreshold time.Duration       // 慢 SQL 查询的阈值
}

// NewZapGormLogger 创建一个新的 ZapGormLogger 实例。
// zapLogger: 传入的 Zap Logger 实例 (通常是 *zap.Logger)。
// cfg: Viper 配置实例，用于读取 GORM 相关的日志配置。
func NewZapGormLogger(zapLogger *zap.Logger, cfg *viper.Viper) gormlogger.Interface {
	logLevelStr := cfg.GetString("gorm.log_level") // 从配置读取 GORM 日志级别字符串
	var logLevel gormlogger.LogLevel
	switch strings.ToLower(logLevelStr) { // 转换为小写以进行不区分大小写的比较
	case "silent":
		logLevel = gormlogger.Silent
	case "error":
		logLevel = gormlogger.Error
	case "warn":
		logLevel = gormlogger.Warn
	case "info":
		logLevel = gormlogger.Info
	default:
		logLevel = gormlogger.Info // 默认级别为 Info
		// 如果配置的级别无效，可以考虑记录一条警告，但此时 GORM logger 尚未完全初始化
	}

	slowThresholdMs := cfg.GetInt("gorm.slow_threshold_ms")
	if slowThresholdMs <= 0 {
		slowThresholdMs = 200 // 默认慢 SQL 阈值为 200ms
	}

	return &ZapGormLogger{
		// 使用 zap.AddCallerSkip(3) 或更高，以确保调用者信息指向实际的业务代码而不是 logger 本身。
		// 具体层数可能需要根据实际调用栈调整。
		ZapLogger:     zapLogger.WithOptions(zap.AddCallerSkip(3)),
		LogLevel:      logLevel,
		SlowThreshold: time.Duration(slowThresholdMs) * time.Millisecond,
	}
}

// LogMode 设置日志级别。GORM 会调用此方法来改变日志记录的详细程度。
func (l *ZapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info 打印普通级别的日志信息。
func (l *ZapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Sugar().Infof(msg, data...)
	}
}

// Warn 打印警告级别的日志信息。
func (l *ZapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.ZapLogger.Sugar().Warnf(msg, data...)
	}
}

// Error 打印错误级别的日志信息。
func (l *ZapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ZapLogger.Sugar().Errorf(msg, data...)
	}
}

// Trace 打印 SQL、行号等 GORM 跟踪信息。
func (l *ZapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows), // GORM v2 返回的是影响行数，对于查询可能是 -1 或 0
		zap.String("sql", sql),
	}

	if err != nil && err != gorm.ErrRecordNotFound { // ErrRecordNotFound 通常不被视为服务端错误
		l.ZapLogger.Error("trace", append(fields, zap.Error(err))...)
		return
	}

	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		l.ZapLogger.Warn("trace_slow_sql", fields...)
		return
	}

	if l.LogLevel >= gormlogger.Info {
		l.ZapLogger.Info("trace", fields...)
	}
}

// ProvideGormDB 是一个 Wire Provider 函数，用于初始化并返回 GORM DB 实例和相关的清理函数。
// cfg: Viper 配置实例，用于读取数据库和 GORM 的配置。
// zapLogger: Zap SugaredLogger 实例，用于此函数内部的日志记录，并传递给 GORM Logger 适配器。
func ProvideGormDB(cfg *viper.Viper, zapSugaredLogger *zap.SugaredLogger) (*gorm.DB, func(), error) {
	driver := cfg.GetString("database.driver")
	if driver != "mysql" {
		return nil, nil, fmt.Errorf("不支持的数据库驱动: %s。当前仅支持 'mysql'", driver)
	}

	dsn := fmt.Sprintf(cfg.GetString("database.source_template"),
		cfg.GetString("database.user"),
		cfg.GetString("database.password"),
		cfg.GetString("database.host"),
		cfg.GetString("database.port"),
		cfg.GetString("database.dbname"),
		cfg.GetString("database.charset"),
		cfg.GetString("database.parseTime"), // 保持为字符串 "True"
	)

	// 使用 SugaredLogger 的底层 Logger (zap.Logger) 创建 GORM Logger 适配器实例。
	gormLogger := NewZapGormLogger(zapSugaredLogger.Desugar(), cfg)

	gormConfig := &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: cfg.GetBool("gorm.skip_default_transaction"),
		PrepareStmt:            cfg.GetBool("gorm.prepare_stmt"),
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		zapSugaredLogger.Errorf("无法连接到数据库 (DSN: %s): %v", dsn, err)
		return nil, nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		zapSugaredLogger.Errorf("获取底层 sql.DB 实例失败: %v", err)
		return nil, nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.GetInt("database.max_idle_conns"))
	sqlDB.SetMaxOpenConns(cfg.GetInt("database.max_open_conns"))
	sqlDB.SetConnMaxLifetime(cfg.GetDuration("database.conn_max_lifetime"))

	cleanup := func() {
		zapSugaredLogger.Info("正在关闭数据库连接...")
		if err := sqlDB.Close(); err != nil {
			zapSugaredLogger.Errorf("关闭数据库连接失败: %v", err)
		} else {
			zapSugaredLogger.Info("数据库连接已成功关闭。")
		}
	}
	zapSugaredLogger.Info("数据库连接成功建立并通过 GORM 初始化。")
	return db, cleanup, nil
}

// ProviderSet 是 db 包的 Wire Provider Set。
// 它导出了 ProvideGormDB 函数，使其可用于 Wire 的依赖注入图谱。
var ProviderSet = wire.NewSet(ProvideGormDB)

// 确保 ZapGormLogger 实现了 gormlogger.Interface (编译时检查)
var _ gormlogger.Interface = (*ZapGormLogger)(nil)
