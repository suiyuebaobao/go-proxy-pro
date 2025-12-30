package repository

import (
	"go-aiproxy/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitMySQL() error {
	// 关闭GORM的默认日志输出，避免打印到控制台
	db, err := gorm.Open(mysql.Open(config.Cfg.MySQL.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(config.Cfg.MySQL.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.Cfg.MySQL.MaxOpenConns)

	DB = db
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

// CloseMySQL 关闭 MySQL 连接
func CloseMySQL() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
