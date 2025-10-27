package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GORMConfig GORM配置
type GORMConfig struct {
	Host         string
	Port         int
	Database     string
	Username     string
	Password     string
	Charset      string
	LogLevel     logger.LogLevel
	MaxIdleConns int
	MaxOpenConns int
}

// DefaultGORMConfig 默认GORM配置
func DefaultGORMConfig() *GORMConfig {
	return &GORMConfig{
		Host:         "localhost",
		Port:         3306,
		Database:     "idle_server",
		Username:     "root",
		Password:     "123456",
		Charset:      "utf8mb4",
		LogLevel:     logger.Info,
		MaxIdleConns: 10,
		MaxOpenConns: 100,
	}
}

// GORM GORM数据库连接管理
type GORM struct {
	db     *gorm.DB
	config *GORMConfig
}

// NewGORM 创建GORM连接
func NewGORM(config *GORMConfig) (*GORM, error) {
	if config == nil {
		config = DefaultGORMConfig()
	}

	// 首先连接到默认数据库来创建目标数据库
	defaultDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=%s&parseTime=True&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Charset,
	)

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 连接到默认数据库
	tempDB, err := gorm.Open(mysql.Open(defaultDSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL server: %w", err)
	}

	// 创建目标数据库
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", config.Database)
	if err := tempDB.Exec(createDBSQL).Error; err != nil {
		return nil, fmt.Errorf("failed to create database %s: %w", config.Database, err)
	}

	// 获取底层SQL连接来关闭临时连接
	sqlDB, _ := tempDB.DB()
	sqlDB.Close()

	// 构建目标数据库DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		config.Charset,
	)

	// 连接到目标数据库
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err = db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := db.Exec("SELECT 1").Error; err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to GORM MySQL database: %s@%s:%d/%s",
		config.Username, config.Host, config.Port, config.Database)

	gormDB := &GORM{
		db:     db,
		config: config,
	}

	return gormDB, nil
}

// GetDB 获取GORM数据库实例
func (g *GORM) GetDB() *gorm.DB {
	return g.db
}

// Close 关闭数据库连接
func (g *GORM) Close() error {
	sqlDB, err := g.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate 自动迁移表结构
func (g *GORM) AutoMigrate(models ...interface{}) error {
	if err := g.db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}
	log.Println("Database auto migration completed")
	return nil
}

// IsHealthy 检查数据库健康状态
func (g *GORM) IsHealthy() bool {
	if g.db == nil {
		return false
	}

	sqlDB, err := g.db.DB()
	if err != nil {
		return false
	}

	if err := sqlDB.Ping(); err != nil {
		log.Printf("Database health check failed: %v", err)
		return false
	}

	return true
}

// GetStats 获取连接池统计信息
func (g *GORM) GetStats() map[string]interface{} {
	sqlDB, err := g.db.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
	}
}

// BeginTransaction 开始事务
func (g *GORM) BeginTransaction() *gorm.DB {
	return g.db.Begin()
}

// GetConnectionStats 获取连接状态详情
func (g *GORM) GetConnectionStats() map[string]interface{} {
	sqlDB, err := g.db.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
	}
}
