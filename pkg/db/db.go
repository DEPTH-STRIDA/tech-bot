package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config структура для хранения параметров конфигурации базы данных
type Config struct {
	Host     string
	Port     string
	UserName string
	DBName   string
	Password string
	SSLMode  string
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase(conf Config) (*gorm.DB, error) {
	dsn := ""
	if conf.Port == "" {
		dsn = fmt.Sprintf("host=%s user=%s dbname=%s password=%s sslmode=%s", conf.Host, conf.UserName, conf.DBName, conf.Password, conf.SSLMode)
	} else {
		dsn = fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", conf.Host, conf.Port, conf.UserName, conf.DBName, conf.Password, conf.SSLMode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
