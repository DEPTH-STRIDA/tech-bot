package db

import (
	"easycodeapp/internal/config"
	m "easycodeapp/internal/model"
	"easycodeapp/pkg/db"
	"easycodeapp/pkg/model"

	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	var err error

	DB, err = db.NewDatabase(db.Config{
		Host:     config.File.DataBaseConfig.Host,
		Port:     config.File.DataBaseConfig.Port,
		UserName: config.File.DataBaseConfig.UserName,
		DBName:   config.File.DataBaseConfig.DBName,
		Password: config.File.DataBaseConfig.Password,
		SSLMode:  config.File.DataBaseConfig.SSLMode,
	})
	if err != nil {
		panic(err)
	}

	DB.AutoMigrate(&model.Form{}, &model.User{}, &m.Mailing{}, &m.MailingStatus{}, &m.Message{}, &m.CachedMessage{})
}
