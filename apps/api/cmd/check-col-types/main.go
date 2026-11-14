package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(os.Getenv("KUN_DATABASE_URL")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}

	type ColInfo struct {
		TableName  string `gorm:"column:table_name"`
		ColumnName string `gorm:"column:column_name"`
		DataType   string `gorm:"column:data_type"`
	}
	var cols []ColInfo
	db.Raw(`SELECT table_name, column_name, data_type FROM information_schema.columns
		WHERE (table_name = 'galgame_engine' AND column_name = 'alias')
		   OR (table_name = 'galgame_rating' AND column_name = 'galgame_type')
		   OR (table_name = 'galgame_toolset' AND column_name = 'homepage')
		   OR (table_name = 'galgame_website' AND column_name = 'domain')
		ORDER BY table_name`).Scan(&cols)

	fmt.Println("当前数据库列类型:")
	for _, c := range cols {
		fmt.Printf("  %-30s %-20s %s\n", c.TableName, c.ColumnName, c.DataType)
	}

	type Migration struct {
		Name      string `gorm:"column:name"`
		AppliedAt string `gorm:"column:applied_at"`
	}
	var migs []Migration
	db.Raw(`SELECT name, to_char(applied_at, 'YYYY-MM-DD HH24:MI:SS') as applied_at FROM _migrations ORDER BY id`).Scan(&migs)

	fmt.Println("\n已应用的迁移:")
	for _, m := range migs {
		fmt.Printf("  %s  (%s)\n", m.Name, m.AppliedAt)
	}
}
