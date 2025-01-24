package model

import (
	"time"

	"gorm.io/gorm"
)

// DefaultForm форма без каких-либо статусов. Содержит только поля, которые заполнял преподаватель
type DefaultForm struct {
	LessonDate      string `json:"lesson-date" gorm:"type:text;column:lesson_date"`
	LessonTime      string `json:"lesson-time" gorm:"type:text;column:lesson_time"`
	ReplaceFormat   string `json:"replace-format" gorm:"type:text;column:replace_format"`
	GroupNumber     string `json:"group-number" gorm:"type:text;column:group_number"`
	Teacher         string `json:"teacher" gorm:"type:text;column:teacher"`
	Subject         string `json:"subject" gorm:"type:text;column:subject"`
	Module          string `json:"module" gorm:"type:text;column:module"`
	Lesson          string `json:"lesson" gorm:"type:text;column:lesson"`
	Reason          string `json:"reason" gorm:"type:text;column:reason"`
	ReplaceTransfer string `json:"replace-transfer" gorm:"type:text;column:replace_transfer"`
	Link            string `json:"link" gorm:"type:text;column:link"`
	ImpInfo         string `json:"imp-info" gorm:"type:text;column:imp_info"`
	ImpInfo2        string `json:"imp-info2" gorm:"type:text;column:imp_info2"`
	MentoringInf1   string `json:"mentoring-inf-1" gorm:"type:text;column:mentoring_inf_1"`
	MentoringInf2   string `json:"mentoring-inf-2" gorm:"type:text;column:mentoring_inf_2"`
	MentoringInf3   string `json:"mentoring-inf-3" gorm:"type:text;column:mentoring_inf_3"`
	TransferTime    string `json:"transfer-time" gorm:"type:text;column:transfer_time"`
	TeamLeader      string `json:"team-leader" gorm:"type:text;column:team_leader"`
}

// DeleteForm форма, по которой производится удаление
type DeleteForm struct {
	ID       int64  `json:"ID"`       // ID заявки
	InitData string `json:"initData"` // Данные телеграмм
}

// BaseModel собственная базовая модель с ID как int64
type BaseModel struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"ID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Form struct {
	BaseModel
	DefaultForm
	Comment               string `json:"comment" gorm:"type:text;column:comment"`
	TelegramUserID        int64  `json:"-" gorm:"column:telegram_id"`                    // ID телеграма
	IsEmergency           bool   `json:"is-emergency" gorm:"is_emergency"`               // Срочная ли заявка
	ReplaceTgStatus       bool   `json:"replace-tg-status" gorm:"replace_tg_status"`     // Дошло ли в обычный тг чат сообщение
	ReplaceMsgId          int    `json:"-" gorm:"replace_msg_id"`                        // ID сообщения в обычном чате
	EmergencyTgStatus     bool   `json:"emergency-tg-status" gorm:"emergency_tg_status"` // Дошло ли в чат спецназ сообщение
	EmergencyMsgId        int    `json:"-" gorm:"emergency_msg_id"`                      // ID сообщения в чате спецназ
	GoogleSheetStatus     bool   `json:"google-sheet-status" gorm:"google_sheet_status"` // Дошла ли заявка в гугл таблицу
	GoogleSheetLineNumber int    `json:"-" gorm:"google_sheet_line_number"`              // Номер строки в гугл таблицах
	ActiveMembers         int    `json:"-" gorm:"column:active_members"`                 // Количество активных учеников группы
	ClickhouseStatus      int    `json:"-" gorm:"clickhouse_status"`
}
