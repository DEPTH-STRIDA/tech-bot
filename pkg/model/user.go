package model

import "gorm.io/gorm"

type User struct {
	gorm.Model

	CRMID       int64  `gorm:"type:bigint" json:"crm_id"`
	TeacherName string `gorm:"type:varchar(255);unique" json:"teacher_name"`
	UserName    string `gorm:"type:varchar(255);unique" json:"username"`
	ChatID      int64  `gorm:"type:bigint" json:"chat_id"`
	UserID      int64  `gorm:"type:bigint" json:"user_id"`
}

// TableName задает имя таблицы для модели User
func (User) TableName() string {
	return "users"
}
