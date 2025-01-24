package web

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/tg"
	"easycodeapp/pkg/model"
	"encoding/json"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleError обрабатывает ошибки, возникающие при взаимодействии с Telegram или базой данных, и отправляет сообщение об ошибке в указанный чат.
func (app *WebApp) handleError(chatName, toTgString string, err error, action string) {
	chat := ""
	switch chatName {
	case ReplaceChat:
		chat = "замен/переносов"
	case EmergencyChat:
		chat = "спецназ"
	case NewChat:
		chat = "новый канал"
	case "Database":
		chat = "база данных"
	default:
		chat = "неизвестный чат"
	}

	// Формирование сообщения об ошибке
	errMsg := fmt.Sprintf("Не удалось %s в %s.\n<strong>Форма:</strong>\n%s\n<strong>Ошибка:</strong>\n%s", action, chat, toTgString, err.Error())
	logger.Info(errMsg)

	// Отправка сообщения об ошибке в Telegram
	newMsg := tgbotapi.NewMessage(config.File.TelegramConfig.BotTgChat, errMsg)
	newMsg.ParseMode = "html"
	newMsg.ReplyToMessageID = config.File.TelegramConfig.ErrorTopicID

	tg.TelegramBot.SendMessageRepetLowPriority(newMsg, 3)
}

// updateDatabase обновляет запись в базе данных с новыми значениями полей.
func (app *WebApp) updateDatabase(f model.Form, updatedFields map[string]interface{}, toTgString string) error {
	logger.Info("Начало обновления базы данных для формы ID ", f.ID)
	logger.Debug("Обновляемые поля: ", updatedFields)

	// Логирование перед вызовом обновления
	logger.Debug("Вызов db.UpdateFormByTelegramID с TelegramID: ", f.TelegramUserID)

	err := db.UpdateFormByTelegramID(f.TelegramUserID, updatedFields)
	if err != nil {
		logger.Error("Ошибка обновления формы в базе данных для TelegramID ", f.TelegramUserID, ": ", err)

		// Использование handleError для обработки ошибки обновления базы данных
		app.handleError("Database", toTgString, err, "обновить статус заявки в базе данных")
		return err
	}

	logger.Info("Успешное обновление базы данных для формы ID ", f.ID)
	logger.Debug("Обновленные поля для формы ID ", f.ID, ": ", updatedFields)

	return nil
}

// SendToChat отправляет сообщение в указанный чат Telegram и обновляет статус в базе данных.
func (app *WebApp) SendToChat(f model.Form, date string, chatName string, GroupId int64) {
	logger.Info("Отправка заявки в чат: ", chatName)

	toTgString := app.PrepareTgMsg(f, date)
	msg := tgbotapi.NewMessage(GroupId, toTgString)

	// Отправка сообщения в Telegram
	sendedMsg, err := tg.TelegramBot.SendMessageRepet(msg, config.File.WebConfig.NumberRepetitions)
	if err != nil {
		app.handleError(chatName, toTgString, err, "отправить сообщение")
		return
	}

	// Подготовка полей для обновления в базе данных
	updatedFields := make(map[string]interface{})
	if chatName == EmergencyChat {
		updatedFields["emergency_tg_status"] = true
		updatedFields["emergency_msg_id"] = sendedMsg.MessageID
	}
	if chatName == ReplaceChat {
		updatedFields["replace_tg_status"] = true
		updatedFields["replace_msg_id"] = sendedMsg.MessageID
	}

	// Обновление записи в базе данных
	db.UpdateFormByID(f.ID, updatedFields)
}

// EditInChat редактирует существующее сообщение в чате Telegram и обновляет статус в базе данных.
func (app *WebApp) EditInChat(f model.Form, chatName, toTgString string, chatID int64, msgID int) {
	msgToEdit := tgbotapi.NewEditMessageText(chatID, msgID, toTgString)

	// Редактирование сообщения в Telegram
	_, err := tg.TelegramBot.EditMessageRepet(msgToEdit, config.File.WebConfig.NumberRepetitions)
	if err != nil {
		app.handleError(chatName, toTgString, err, "редактировать данные")
		return
	}

	// Подготовка полей для обновления в базе данных
	updatedFields := make(map[string]interface{})
	if chatName == EmergencyChat {
		updatedFields["emergency_tg_status"] = true
	}
	if chatName == ReplaceChat {
		updatedFields["replace_tg_status"] = true
	}

	// Обновление записи в базе данных
	db.UpdateFormByID(f.ID, updatedFields)
}

// DeleteInChat удаляет сообщение из чата Telegram и обновляет статус в базе данных.
func (app *WebApp) DeleteInChat(f model.Form, chatName string, chatID int64, msgID int) {
	msgToDelete := tgbotapi.NewDeleteMessage(chatID, msgID)

	// Удаление сообщения в Telegram
	err := tg.TelegramBot.DeleteMessageRepet(msgToDelete, config.File.WebConfig.NumberRepetitions)
	if err != nil {
		formJson, jsonErr := json.Marshal(f)
		toTgString := ""
		if jsonErr != nil {
			toTgString = fmt.Sprintf("Ошибка сериализации формы: %s", jsonErr.Error())
		} else {
			toTgString = string(formJson)
		}

		// Использование handleError для обработки ошибки удаления сообщения
		app.handleError(chatName, toTgString, err, "удалить сообщение")
		return
	}

	// Подготовка полей для обновления в базе данных
	updatedFields := make(map[string]interface{})
	if chatName == EmergencyChat {
		updatedFields["emergency_tg_status"] = true
		updatedFields["emergency_msg_id"] = 0
	}
	if chatName == ReplaceChat {
		updatedFields["replace_tg_status"] = true
		updatedFields["replace_msg_id"] = 0
	}

	// Обновление записи в базе данных
	db.UpdateFormByID(f.ID, updatedFields)
}
