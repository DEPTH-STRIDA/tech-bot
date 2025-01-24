package tg

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var defaultHandler = Handler{
	Func: func(app *Bot, update tgbotapi.Update) error {
		return nil
	},
	Description: "defaultHandler",
}

type HandlerFunc func(*Bot, tgbotapi.Update) error

type Handler struct {
	Func        HandlerFunc
	Description string
}

type State struct {
	Global            bool // Будут проверяться триггеры несмотря нахождение пользователя в другом "СОСТОЯНИИ"
	NotEntranceAction bool // Отключить ли действие при входе
	NoContext         bool // Не переключать ли состояние при входе
	CatchAll          bool // Содержится ли функция для обработки любых сообщений
	CatchAllCallBack  bool // Содержится ли функция для обработки  любых callback'ов

	AtEntranceFunc       Handler
	CatchAllFunc         Handler
	CatchAllCallBackfunc Handler

	MessageRoute  map[string]Handler
	CallBackRoute map[string]HandlerFunc
}

type CallBackAction struct {
	ActionType      string `json:"ActionType"`
	MailingID       int64  `json:"MailingID"`
	StatusID        int64  `json:"StatusID"`
	NotificationUID string `json:"NotificationUID"`
	UpdateType      string `json:"updateType"`
}
