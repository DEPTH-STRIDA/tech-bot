package model

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID                       uint      `gorm:"primaryKey"`
	LessonID                 int64     `gorm:"type:bigint"`          // ID урока
	CRMID                    int64     `gorm:"type:bigint;not null"` // Необходимо хранить, чтобы в случае обновления установить нового владельца сообщения
	ChatID                   int64     `gorm:"type:bigint;not null"` // ID тг чата
	ChatName                 string    `gorm:"type:varchar(255)"`
	UserName                 string    `gorm:"type:varchar(255);not null"`
	MsgID                    int64     `gorm:"type:bigint"`
	UID                      string    `gorm:"type:varchar(255);not null"` // ID сообщения
	MsgSendTime              time.Time `gorm:"type:timestamp"`             // Время, в которое надо отправить сообщение
	DelayTime                time.Time `gorm:"type:timestamp"`             // Время, в которое сообщение считается просроченным
	MsgIssent                bool      `gorm:"type:boolean;not null"`      // Отправлено ли  сообщение
	MsgIsPressed             bool      `gorm:"type:boolean;not null"`      // Была ли реакция на сообщение от пользователя
	IsReacted                bool      `gorm:"type:boolean;not null"`      // Было ли переслано ли это сообщение в чат админам
	MsgIsMorningNotification bool      `gorm:"type:boolean;not null"`      // Сообщение утреннее уведомление

	TeacherName  string    `gorm:"type:varchar(255)"`          // ФИО преподавателя
	CourseName   string    `gorm:"type:varchar(255);not null"` // Название курса
	CourseID     int       `gorm:"type:integer"`               // Номер курса (группы)
	LessonTime   time.Time `gorm:"type:timestamp"`             // Дата и время урока
	ActiveMember int       `gorm:"type:integer"`               // Количество активных учеников
	LessonNumber int       `gorm:"type:integer"`               // Номер урока
}

type CachedMessage struct {
	gorm.Model
	Message
}
