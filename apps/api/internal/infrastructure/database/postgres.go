package database

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"kun-galgame-api/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func withTimeZone(dsn string) string {
	if strings.Contains(dsn, "TimeZone=") || strings.Contains(dsn, "timezone=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "TimeZone=Asia/Shanghai"
}

func NewPostgres(cfg config.DatabaseConfig, mode string) *gorm.DB {
	logLevel := logger.Warn
	if mode == "dev" {
		logLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(postgres.Open(withTimeZone(cfg.URL)), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("获取数据库连接池失败: %v", err))
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	slog.Info("数据库连接成功")
	return db
}

// Quiet is a session whose statements stay out of the dev terminal unless they
// are slow or fail. Dev logs every statement, and an idle `pnpm dev` on
// 2026-11-02 filled the screen with the rating mirror re-applying an unchanged
// partial page and the catalog verify window's id list.
func Quiet(db *gorm.DB) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.Session(&gorm.Session{Logger: db.Logger.LogMode(logger.Warn)})
}
