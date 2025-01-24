package model

import (
	"gorm.io/gorm"
)

// Mailing основная таблица для хранения информации о рассылках
type Mailing struct {
	gorm.Model

	AuthorTgID      int64           `gorm:"column:author_tg_id;not null"`                     // ID пользователя Telegram, создавшего рассылку
	MailingType     string          `gorm:"column:mailing_type;not null"`                     // Тип рассылки
	CohortName      string          `gorm:"column:cohort_name;not null"`                      // Имя когорты, которой предназначена рассылка
	MessageText     string          `gorm:"column:message_text;not null"`                     // Текст сообщения для рассылки
	Button          bool            `gorm:"column:button;not null"`                           // Нужна ли кнопка подтверждения в сообщении
	MailingFinished bool            `gorm:"column:mailing_finished;not null;default:false"`   // Завершена ли отправка всех сообщений
	MailingExpired  bool            `gorm:"column:mailing_expired;not null;default:false"`    // Истек ли срок действия рассылки. Нужно ли проверять просрочевших пользователей.
	MailingStatuses []MailingStatus `gorm:"foreignKey:MailingID;constraint:OnDelete:CASCADE"` // Связь один-ко-многим со статусами отправки
	Entities        string          `gorm:"type:jsonb;default:null"`
}

// MailingStatus таблица для хранения статусов отправки сообщений каждому пользователю
type MailingStatus struct {
	gorm.Model
	MailingID    uint   `gorm:"column:mailing_id;not null"`                   // ID рассылки, к которой относится статус
	UserName     string `gorm:"column:user_name;not null"`                    // Имя пользователя (чата) куда отправляется сообщение
	TgID         int64  `gorm:"column:tg_id;not null"`                        // ID пользователя Telegram, которому отправляется сообщение
	MsgIsSent    bool   `gorm:"column:msg_is_sent;not null;default:false"`    // Было ли отправлено сообщение
	MsgID        int    `gorm:"column:msg_id;default:null"`                   // ID отправленного сообщения в Telegram
	SendFailed   bool   `gorm:"column:send_failed;default:null"`              // Отправка не удалась
	MsgIsReacted bool   `gorm:"column:msg_is_reacted;not null;default:false"` // Отреагировал ли пользователь на сообщение

}
