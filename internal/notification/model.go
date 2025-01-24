package notification

import "easycodeapp/internal/model"

type OldMessage struct {
	model.Message
}

type CallBack struct {
	ID       uint   `gorm:"primaryKey"`
	UID      string `gorm:"type:varchar(255);unique;not null"`
	MsgID    int64  `gorm:"type:bigint;not null"`
	ChatID   int64  `gorm:"type:bigint;not null"`
	UserName string `gorm:"type:varchar(255);not null"`
}
