package db

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap" // 导入 zap.SugaredLogger
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/google/wire" // 导入 Wire 包
)

// ProviderSet 是 db 包的 Wire Provider Set。
// 它导出了 NewGormDB 函数，使其可用于 Wire 的依赖注入图谱。
var ProviderSet = wire.NewSet(NewGormDB)

// zapGormWriter 是一个适配器，它实现了 gormlogger.Writer 接口，
// 并将 GORM 的日志消息路由到 Zap SugaredLogger。
type zapGormWriter struct {
	SugaredLogger *zap.SugaredLogger
}

// Printf 实现了 gormlogger.Writer 接口的 Printf 方法。
// 它使用 Zap SugaredLogger 的 Infof 方法来记录格式化的日志消息。
// GORM 通常使用 Printf 来输出其日志。
func (w *zapGormWriter) Printf(format string, args ...interface{}) {
	w.SugaredLogger.Infof(format, args...)
}

// NewGormDB 函数根据 Viper 配置初始化并返回一个新的 GORM数据库连接实例。
// 它现在接收 Viper 实例和 Zap SugaredLogger 实例作为参数，用于依赖注入。
// 返回 GORM DB 实例，一个用于关闭连接的清理函数，以及可能发生的错误。
func NewGormDB(cfg *viper.Viper, zapLog *zap.SugaredLogger) (*gorm.DB, func(), error) {
	// 从 Viper 配置实例中读取数据库配置信息
	host := cfg.GetString("database.host")
	port := cfg.GetString("database.port")
	user := cfg.GetString("database.user")
	password := cfg.GetString("database.password")
	dbname := cfg.GetString("database.dbname")
	charset := cfg.GetString("database.charset")
	parseTime := cfg.GetBool("database.parseTime")

	// 构造 DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=%t&loc=Local",
		user, password, host, port, dbname, charset, parseTime,
	)

	// 配置 GORM 日志记录器
	// 使用 zapGormWriter 将 GORM 日志输出到 Zap
	gormZapLogger := gormlogger.New(
		&zapGormWriter{SugaredLogger: zapLog}, // 使用自定义的 Zap writer
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond, // 慢 SQL 阈值，例如 200ms
			LogLevel:                  gormlogger.Warn,        // 默认日志级别 (Warn, Error, Info, Silent)
			IgnoreRecordNotFoundError: true,                   // 忽略 ErrRecordNotFound 错误，避免不必要的告警
			Colorful:                  false,                  // 生产环境通常禁用彩色打印
		},
	)

	// 根据应用环境调整 GORM 日志级别
	appEnv := cfg.GetString("APP_ENV") // 从 Viper 获取 APP_ENV
	if appEnv != "production" {
		gormZapLogger = gormZapLogger.LogMode(gormlogger.Info) // 开发环境使用更详细的 Info 级别
		zapLog.Info("GORM 日志级别设置为 Info (非生产环境)")
	} else {
		zapLog.Info("GORM 日志级别设置为 Warn (生产环境)")
	}

	// 使用 GORM 打开 MySQL 连接
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormZapLogger, // 使用配置好的、集成了 Zap 的 GORM logger
		// NamingStrategy: schema.NamingStrategy{ // 可选：自定义命名策略
		//	TablePrefix:   "t_", // 表名前缀
		//	SingularTable: true, // 使用单数表名
		// },
		// DisableForeignKeyConstraintWhenMigrating: true, // 迁移时不创建外键约束
	})

	if err != nil {
		zapLog.Errorf("连接 MySQL 数据库失败: %v", err)
		return nil, nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	// 获取底层的 sql.DB 实例以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		zapLog.Errorf("获取底层 sql.DB 实例失败: %v", err)
		// 即使无法获取 sql.DB，db 实例本身可能仍然可用，但无法配置连接池。
		// 决定是否返回错误取决于应用需求。这里我们选择返回错误。
		return nil, nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	// 配置连接池参数
	// 从 Viper 配置中读取连接池设置，如果未设置则使用合理的默认值。
	maxIdleConns := cfg.GetInt("database.max_idle_conns")
	if maxIdleConns == 0 {
		maxIdleConns = 10 // 默认值
	}
	sqlDB.SetMaxIdleConns(maxIdleConns)

	maxOpenConns := cfg.GetInt("database.max_open_conns")
	if maxOpenConns == 0 {
		maxOpenConns = 100 // 默认值
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)

	connMaxLifetimeHours := cfg.GetDuration("database.conn_max_lifetime_hours")
	if connMaxLifetimeHours == 0 {
		connMaxLifetimeHours = time.Hour // 默认值
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetimeHours)

	zapLog.Infof("成功连接到 MySQL 数据库! (MaxIdleConns: %d, MaxOpenConns: %d, ConnMaxLifetime: %v)",
		maxIdleConns, maxOpenConns, connMaxLifetimeHours)

	// 定义清理函数，用于关闭数据库连接
	cleanup := func() {
		zapLog.Info("正在关闭数据库连接...")
		if err := sqlDB.Close(); err != nil {
			zapLog.Errorf("关闭数据库连接失败: %v", err)
		} else {
			zapLog.Info("数据库连接已成功关闭。")
		}
	}

	return db, cleanup, nil
}
