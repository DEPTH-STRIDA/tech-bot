package tg

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	"fmt"
	"strconv"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (app *Bot) SendMessageUnkownChatIdD(msg tgbotapi.MessageConfig) (int64, tgbotapi.Message, error) {
	originalChatId := msg.ChatID
	var sendedMsg tgbotapi.Message
	var err error

	logger.Info("Начинаем попытку отправки сообщения",
		"originalChatId", originalChatId,
		"messageText", msg.Text[:min(len(msg.Text), 50)]) // Логируем первые 50 символов сообщения

	// Попытка 1: отправка с оригинальным ID
	logger.Info("Попытка 1: Отправка с оригинальным ID", "chatId", msg.ChatID)
	sendedMsg, err = app.SendMessage(msg)
	if err == nil {
		logger.Info("Успешная отправка с оригинальным ID",
			"chatId", msg.ChatID,
			"messageId", sendedMsg.MessageID)
		return msg.ChatID, sendedMsg, nil
	}
	logger.Error("Ошибка при отправке с оригинальным ID",
		"chatId", msg.ChatID,
		"error", err)

	// Попытка 2: отправка с префиксом -100
	newChatID := addNegative100Prefix(msg.ChatID)
	logger.Info("Попытка 2: Отправка с префиксом -100",
		"originalChatId", msg.ChatID,
		"newChatId", newChatID)

	msg.ChatID = newChatID
	sendedMsg, err = app.SendMessage(msg)
	if err == nil {
		logger.Info("Успешная отправка с префиксом -100",
			"chatId", msg.ChatID,
			"messageId", sendedMsg.MessageID)
		return msg.ChatID, sendedMsg, nil
	}
	logger.Error("Ошибка при отправке с префиксом -100",
		"chatId", msg.ChatID,
		"error", err)

	// Попытка 3: отправка с отрицательным ID
	negativeChatID := originalChatId * -1
	logger.Info("Попытка 3: Отправка с отрицательным ID",
		"originalChatId", originalChatId,
		"negativeChatId", negativeChatID)

	msg.ChatID = negativeChatID
	sendedMsg, err = app.SendMessage(msg)
	if err == nil {
		logger.Info("Успешная отправка с отрицательным ID",
			"chatId", msg.ChatID,
			"messageId", sendedMsg.MessageID)
		return msg.ChatID, sendedMsg, nil
	}
	logger.Error("Ошибка при отправке с отрицательным ID",
		"chatId", msg.ChatID,
		"error", err)

	// Все попытки неудачны
	logger.Error("Все попытки отправки сообщения неудачны",
		"originalChatId", originalChatId,
		"lastTriedChatId", msg.ChatID,
		"finalError", err)

	return msg.ChatID, sendedMsg, fmt.Errorf("failed all attempts to send message: %w", err)
}

func addNegative100Prefix(num int64) int64 {
	str := fmt.Sprintf("-100%d", num)
	result, _ := strconv.ParseInt(str, 10, 64)
	return result
}

// SendMessage синхронная функция для отправки сообщения
func (app *Bot) SendDeleteMessage(msg tgbotapi.DeleteMessageConfig) (*tgbotapi.APIResponse, error) {
	sendedMsg, err := app.sendDeleteMessage(msg)
	if err != nil {
		return sendedMsg, err
	}
	return sendedMsg, nil
}

// sendMessage асинхронная функция, которая с помощью waitgroup дожидается результатов от отправки сообщения
func (app *Bot) sendDeleteMessage(msg tgbotapi.DeleteMessageConfig) (*tgbotapi.APIResponse, error) {
	var sendedMsg *tgbotapi.APIResponse
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		sendedMsg, err = app.botAPI.Request(msg)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return sendedMsg, err
}

// SendMessageRepet делает несколько попыток отправки сообщений.
// Останавливает попытки после первой успешной.
func (app *Bot) SendMessageRepet(msg tgbotapi.MessageConfig, numberRepetion int) (tgbotapi.Message, error) {
	for i := 0; i < numberRepetion; i++ {
		sendedMsg, err := app.SendMessage(msg)
		if err != nil {
			logger.Info("Ошибка при отправке сообщения с повтором (", i, "):  ", err)
		} else {
			return sendedMsg, nil
		}
	}
	return tgbotapi.Message{}, fmt.Errorf("ни одна попытка не оказалось результативной")
}

// SendMessage синхронная функция для отправки сообщения
func (app *Bot) SendMessage(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	sendedMsg, err := app.sendMessage(msg)
	if err != nil {
		return sendedMsg, err
	}
	return sendedMsg, nil
}

// sendMessage асинхронная функция, которая с помощью waitgroup дожидается результатов от отправки сообщения
func (app *Bot) sendMessage(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	var sendedMsg tgbotapi.Message
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		sendedMsg, err = app.botAPI.Send(msg)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return sendedMsg, err
}

// SendPinMessageEvent синхронная функция для отправки события на закрепление сообщения
func (app *Bot) SendPinMessageEvent(messageID int, ChatID int64, disableNotification bool) (*tgbotapi.APIResponse, error) {
	APIResponse, err := app.sendPinMessageEvent(messageID, ChatID, disableNotification)
	if err != nil {
		return APIResponse, err
	}
	return APIResponse, nil
}

// sendPinMessageEvent асинхронная функция, которая с помощью waitgroup дожидается результатов закрепления сообщения
// DisableNotification - если true, уведомление о закреплении не будет отправлено
func (app *Bot) sendPinMessageEvent(messageID int, ChatID int64, disableNotification bool) (*tgbotapi.APIResponse, error) {
	var APIResponse *tgbotapi.APIResponse
	var err error

	// Закрепление отправленного сообщения
	pinConfig := tgbotapi.PinChatMessageConfig{
		ChatID:              ChatID,
		MessageID:           messageID,
		DisableNotification: disableNotification,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		APIResponse, err = app.botAPI.Request(pinConfig)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return APIResponse, err
}

// SendSticker синхронная функция для отправки стикера
func (app *Bot) SendSticker(stickerID string, chatID int64) (tgbotapi.Message, error) {
	sendedMsg, err := app.sendSticker(stickerID, chatID)
	if err != nil {
		return sendedMsg, err
	}
	return sendedMsg, nil
}

// sendSticker асинхронная функция, которая с помощью waitgroup дожидается результатов от отправки стикера
func (app *Bot) sendSticker(stickerID string, chatID int64) (tgbotapi.Message, error) {
	var sendedMsg tgbotapi.Message
	var err error

	msg := tgbotapi.NewSticker(chatID, tgbotapi.FileID(stickerID))

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		sendedMsg, err = app.botAPI.Send(msg)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return sendedMsg, err
}

// SendUnPinAllMessageEvent синхронная функция для отправки события на открепление всех сообщений.
func (app *Bot) SendUnPinAllMessageEvent(ChannelUsername string, chatID int64) (*tgbotapi.APIResponse, error) {
	sendedMsg, err := app.sendUnPinAllMessageEvent(ChannelUsername, chatID)
	if err != nil {
		return sendedMsg, err
	}
	return sendedMsg, nil
}

// sendUnPinAllMessageEvent асинхронная функция, которая с помощью waitgroup дожидается результатов от отправки события открепления всех сообщений
func (app *Bot) sendUnPinAllMessageEvent(ChannelUsername string, chatID int64) (*tgbotapi.APIResponse, error) {
	var APIresponse *tgbotapi.APIResponse
	var err error

	unpinConfig := tgbotapi.UnpinAllChatMessagesConfig{
		ChatID:          chatID,
		ChannelUsername: ChannelUsername,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		APIresponse, err = app.botAPI.Request(unpinConfig)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return APIresponse, err
}

func (app *Bot) EditMessageRepet(editMsg tgbotapi.EditMessageTextConfig, numberRepetion int) (*tgbotapi.APIResponse, error) {
	var err error
	var response *tgbotapi.APIResponse

	for i := 0; i < numberRepetion; i++ {
		response, err = app.editMessage(editMsg)
		if err != nil {
			logger.Info("Ошибка при редактировании сообщения с повтором (", i, "):  ", err)
		} else {
			return response, nil
		}
	}
	return nil, fmt.Errorf("ни одна попытка не стала результативной: %w", err)
}

// EditMessage синхронно редактирует сообщение
func (app *Bot) EditMessage(editMsg tgbotapi.EditMessageTextConfig) (*tgbotapi.APIResponse, error) {
	response, err := app.editMessage(editMsg)
	if err != nil {
		return response, err
	}

	return response, nil
}

// editMessage редактирует сообщение в чате, отправив функцию редактирования в запросы
func (app *Bot) editMessage(editMsg tgbotapi.EditMessageTextConfig) (*tgbotapi.APIResponse, error) {
	var response *tgbotapi.APIResponse
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		response, err = app.botAPI.Request(editMsg)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return response, err
}

func (app *Bot) DeleteMessageRepet(msgToDelete tgbotapi.DeleteMessageConfig, numberRepetion int) error {
	var err error

	for i := 0; i < numberRepetion; i++ {
		err = app.deleteMessage(msgToDelete)
		if err != nil {
			logger.Info("Не удалось удалить сообщение из чата. Попытка: ", i, " err: ", err)
		} else {
			return nil
		}
	}

	return fmt.Errorf("ни одна попытка не стала результативной: %w", err)
}

// DeleteMessage удаляет сообщение
func (app *Bot) DeleteMessage(msgToDelete tgbotapi.DeleteMessageConfig) error {
	err := app.deleteMessage(msgToDelete)
	if err != nil {
		return err
	}

	return nil
}

func (app *Bot) deleteMessage(deleteMsg tgbotapi.DeleteMessageConfig) error {
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleRequest(func() error {
		defer wg.Done()

		_, err = app.botAPI.Request(deleteMsg)
		if err != nil {
			return err
		}

		return nil
	})

	wg.Wait()
	return err
}

// ShowAlert показывает пользователю предупреждение. alert по типу браузерного.
// Для закрытия такого уведомления потребуется нажать "ок"
func (app *Bot) ShowAlert(CallbackQueryID string, alertText string) {
	callback := tgbotapi.NewCallback(CallbackQueryID, alertText)
	// Это заставит текст появиться во всплывающем окне
	callback.ShowAlert = true
	_, err := app.botAPI.Request(callback)
	if err != nil {
		logger.Info("Не удалось показать alert после CallbackQuery: ", err)
	}
}

func CreateKeyboard(input []string, buttonsPerRow int) tgbotapi.ReplyKeyboardMarkup {
	var keyboard [][]tgbotapi.KeyboardButton

	for i := 0; i < len(input); i += buttonsPerRow {
		var row []tgbotapi.KeyboardButton
		end := i + buttonsPerRow
		if end > len(input) {
			end = len(input)
		}
		for _, text := range input[i:end] {
			row = append(row, tgbotapi.NewKeyboardButton(text))
		}
		keyboard = append(keyboard, row)
	}

	return tgbotapi.NewReplyKeyboard(keyboard...)
}

type ButtonData struct {
	Text string
	Data string
}

//	buttons := [][]telegram.ButtonData{
//		{
//			{Text: "1.com", Data: "http://1.com"},
//			{Text: "2", Data: "2"},
//			{Text: "3", Data: "3"},
//		},
//		{
//			{Text: "4", Data: "4"},
//			{Text: "5", Data: "5"},
//			{Text: "6", Data: "6"},
//		},
//	}
func CreateInlineKeyboard(buttons [][]ButtonData) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, row := range buttons {
		var keyboardRow []tgbotapi.InlineKeyboardButton
		for _, btn := range row {
			keyboardRow = append(keyboardRow, tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.Data))
		}
		keyboard = append(keyboard, keyboardRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

// SendMessageRepetLowPriority синхронная функция, которая отправляет сообщение в телеграм с низким приоритетом
func (app *Bot) SendMessageRepetLowPriority(msg tgbotapi.MessageConfig, numberRepetion int) (tgbotapi.Message, error) {
	for i := 0; i < numberRepetion; i++ {
		sendedMsg, err := app.SendMessageLowPriority(msg)
		if err != nil {
			logger.Info("Ошибка при отправке сообщения с повтором (", i, "):  ", err)
		} else {
			return sendedMsg, nil
		}
	}
	return tgbotapi.Message{}, nil
}

// SendMessageLowPriority синхронная функция, которая отправляет сообщение в телеграм с низким приоритетом
func (app *Bot) SendMessageLowPriority(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	sendedMsg, err := app.sendMessageLowPriority(msg)
	if err != nil {
		return sendedMsg, err
	}
	return sendedMsg, nil
}

// sendMessage асинхронная функция, которая с помощью waitgroup дожидается результатов от отправки сообщения
func (app *Bot) sendMessageLowPriority(msg tgbotapi.MessageConfig) (tgbotapi.Message, error) {
	var sendedMsg tgbotapi.Message
	var err error

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleLowPriorityRequest(func() error {
		defer wg.Done()

		// Устанавливаем глобальные параметры
		sendedMsg, err = app.botAPI.Send(msg)
		if err != nil {
			return err
		}
		return nil

	})

	wg.Wait()
	return sendedMsg, err
}

func (app *Bot) EditMessageRepetLowPriority(editMsg tgbotapi.EditMessageTextConfig, numberRepetion int) (*tgbotapi.APIResponse, error) {
	for i := 0; i < numberRepetion; i++ {
		response, err := app.EditMessageLowPriority(editMsg)
		if err != nil {
			logger.Info("Ошибка при редактировании сообщения с повтором (", i, "):  ", err)
		} else {
			return response, nil
		}
	}
	return nil, nil
}

// EditMessageLowPriority синхронная функция дли редактирования вообщения
func (app *Bot) EditMessageLowPriority(editMsg tgbotapi.EditMessageTextConfig) (*tgbotapi.APIResponse, error) {
	response, err := app.editMessageLowPriority(editMsg)
	if err != nil {
		return response, err
	}

	return response, nil
}

// editMessage редактирует сообщение в чате, отправив функцию редактирования в запросы
func (app *Bot) editMessageLowPriority(editMsg tgbotapi.EditMessageTextConfig) (*tgbotapi.APIResponse, error) {
	var err error
	var editedMsg *tgbotapi.APIResponse

	var wg sync.WaitGroup
	wg.Add(1)

	// Отправляем функцию в канал
	app.msgRequestHandler.HandleLowPriorityRequest(func() error {
		defer wg.Done()

		editedMsg, err = app.botAPI.Request(editMsg)
		if err != nil {
			return err
		}
		return nil
	})

	wg.Wait()
	return editedMsg, err
}

// SendAllAdmins отправляет текст всем админам в лс и ошибку в топик с ошибками
func (app *Bot) SendAllAdmins(msgTexts string) error {
	conf := config.File.TelegramConfig

	// for _, v := range cache.TelegramCacheApp.TgAdminIDS {

	// 	msg := tgbotapi.NewMessage(v, msgTexts)
	// 	msg.ParseMode = "html"
	// 	_, err := app.SendMessage(msg)
	// 	if err != nil {
	// 		logger.Info("Ошибка при отправке сообщения админу (", v, "):  ", err)
	// 	}

	// }

	newMsg := tgbotapi.NewMessage(conf.BotTgChat, msgTexts)
	newMsg.ParseMode = "html"
	newMsg.ReplyToMessageID = conf.ErrorTopicID

	app.SendMessageRepetLowPriority(newMsg, 3)

	return nil
}

// SendQuantityReplaceTransfer отправляет сводку о количестве заявок за день в указанные чаты.
//
// Параметры:
//   - chatIDs: срез идентификаторов чатов, куда нужно отправить сообщение
//   - general: общее количество заявок
//   - replace: количество заявок на замену
//   - transfer: количество заявок на перенос
//
// Функция формирует текстовое сообщение с информацией о количестве заявок
// и отправляет его в каждый из указанных чатов. Если при отправке сообщения
// возникает ошибка, она логируется, но выполнение функции продолжается для
// остальных чатов.
//
// Возвращает nil, так как ошибки обрабатываются внутри функции и не прерывают её работу.
func (app *Bot) SendQuantityReplaceTransfer(chatIDs []int64, general, replace, transfer int64) error {
	msgText := "За день было отправлено\nВсего: " + fmt.Sprint(general) + " заявок\n" + "Замен: " + fmt.Sprint(replace) + "\n" + "Переносов: " + fmt.Sprint(transfer)

	for _, v := range chatIDs {
		msg := tgbotapi.NewMessage(v, msgText)
		_, err := app.SendMessage(msg)
		if err != nil {
			logger.Info("Ошибка при отправке сообщения в чат админов (", v, "):  ", err)
		}
	}
	return nil
}

func (app *Bot) SendMessageButtonLowPriorityRepet(chatID int64, msgText, buttonText, buttonCallbackText string, numberRepetion int) (tgbotapi.Message, error) {
	var err error
	var sendedMsg tgbotapi.Message
	var trueError error

	for i := 0; i < numberRepetion; i++ {
		sendedMsg, err = app.SendMessageButtonLowPriority(chatID, msgText, buttonText, buttonCallbackText)
		if err != nil {
			trueError = err
			logger.Info("Ошибка при отправке сообщения: ", err)
		} else {
			return sendedMsg, nil
		}
	}

	return tgbotapi.Message{}, trueError
}

// SendMessage синхронная функция, которая отправляет сообщение с кнопкой
func (app *Bot) SendMessageButtonLowPriority(chatID int64, msgText, buttonText, buttonCallbackText string) (tgbotapi.Message, error) {

	msg := tgbotapi.NewMessage(chatID, msgText)
	row := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonCallbackText),
	}

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(row)
	msg.ParseMode = "html"

	sendedMsg, err := app.SendMessageLowPriority(msg)

	return sendedMsg, err
}
