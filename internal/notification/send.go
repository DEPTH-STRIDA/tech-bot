package notification

import (
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/model"
	"easycodeapp/internal/tg"
	"errors"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (app *NotificationManager) handleSendMessages(results []model.Message) {
	for _, msg := range results {
		app.sendMessage(msg)
	}
}

func (app *NotificationManager) sendMessage(message model.Message) {
	// 1. Проверяем, что дата урока совпадает с текущей датой
	if !isSameYearMonthDay(time.Now(), message.LessonTime) {
		return
	}

	// 2. Проверяем, что сообщение еще не было отправлено
	if message.MsgIssent {
		return
	}

	// 4. Проверяем, что время урока еще не наступило
	if hasDatePassed(message.LessonTime) {
		// logger.Info(message.LessonTime.Local(), ">", time.Now().Local(), " : ", hasDatePassed(message.MsgSendTime))
		// logger.Info(message.LessonTime, ">", time.Now().Local(), " : ", hasDatePassed(message.MsgSendTime))
		return
	}

	// 3. Проверяем, что время отправки сообщения уже наступило
	if !hasDatePassed(message.MsgSendTime) {

		// logger.Info(message.MsgSendTime.Local(), ">", time.Now().Local(), " : ", hasDatePassed(message.MsgSendTime))
		// logger.Info(message.MsgSendTime, ">", time.Now().Local(), " : ", hasDatePassed(message.MsgSendTime))
		return
	}

	// Время отправления сообщения прошло, и время урока еще не наступило
	data := fmt.Sprintf(`{"ActionType":"Notification","NotificationUID":"%s"}`, message.UID)
	logger.Info("Отправка сообщения для урока UID:", data, "на ChatID:", message.ChatID)

	toTGString := prepareLessonNotification(message)
	chatID := int64(0)
	// Отправка с префиксом "-100"
	sendedMsg, err := tg.TelegramBot.SendMessageButtonLowPriorityRepet(addNegative100Prefix(message.ChatID), toTGString, "✔️ Все хорошо. Я буду на уроке.", data, 3)
	chatID = addNegative100Prefix(message.ChatID)
	if err != nil {

		// Отправка без модификаций
		sendedMsg, err = tg.TelegramBot.SendMessageButtonLowPriorityRepet(message.ChatID, toTGString, "✔️ Все хорошо. Я буду на уроке.", data, 3)
		chatID = message.ChatID
		if err != nil {

			// Отправка с префиксом "-"
			sendedMsg, err = tg.TelegramBot.SendMessageButtonLowPriorityRepet(message.ChatID*-1, toTGString, "✔️ Все хорошо. Я буду на уроке.", data, 3)
			chatID = message.ChatID * -1
			if err != nil {
				logger.Error("Ошибка при отправке сообщения с информацией о группе: ", err, "\n;Сообщение ", data, "; преподавателя: ", message.TeacherName, message.UserName)
				return
			}
		}
	}

	logger.Info("Отправлено сообщения об уведомлении: ", sendedMsg.MessageID, sendedMsg.Text)
	if sendedMsg.MessageID == 0 {
		logger.Error("Сообщение отправилось, но в ответ пришло без ошибки: ", message)
		return
	}

	// Обновляем chatId отправленного сообщения
	message.MsgIssent = true
	updateRowConfig := UpdateRowConfig{
		SearchColumnName: "uid",
		SearchValue:      message.UID,
		NewColumnName:    "chat_id",
		NewValue:         chatID,
		Row:              model.Message{},
	}
	err = app.Message.UpdateRow(updateRowConfig)
	if err != nil {
		logger.Error("Ошибка при обновлении статуса отправленного сообщения: ", err)
		return
	}
	// Обновляем статус отправленного сообщения
	message.MsgIssent = true
	updateRowConfig = UpdateRowConfig{
		SearchColumnName: "uid",
		SearchValue:      message.UID,
		NewColumnName:    "msg_issent",
		NewValue:         true,
		Row:              model.Message{},
	}
	err = app.Message.UpdateRow(updateRowConfig)
	if err != nil {
		logger.Error("Ошибка при обновлении статуса отправленного сообщения: ", err)
		return
	}
	// Обновляем статус отправленного сообщения
	message.MsgIssent = true
	updateRowConfig = UpdateRowConfig{
		SearchColumnName: "uid",
		SearchValue:      message.UID,
		NewColumnName:    "msg_id",
		NewValue:         sendedMsg.MessageID,
		Row:              model.Message{},
	}
	err = app.Message.UpdateRow(updateRowConfig)
	if err != nil {
		logger.Error("Ошибка при обновлении статуса отправленного сообщения: ", err)
		return
	}
	// Добавляем обработчик нажатия кнопки
	app.AddHandlerButtonPress(message, sendedMsg.MessageID)

	logger.Info("Сообщение успешно отправлено и обработчик добавлен для урока UID:", message.UID, "; преподавателя: ", message.TeacherName, message.UserName)
}

// HandlCallbackRoute обрабатывает нажатие на кнопку.
func (app *NotificationManager) HandlCallbackRoute(appBot *tg.Bot, update tgbotapi.Update, UID string) error {
	// Получение callback из БД
	message := CallBack{}
	err := app.CallBack.GetRowByColumn("uid", UID, &message)
	if err != nil {
		logger.Info("Пользователь ", update.CallbackQuery.From.FirstName, " нажал на кнопку под сообщением, которого нет в БД CallBack")
		tg.TelegramBot.ShowAlert(update.CallbackQuery.ID, "Внутренняя ошибка")
		return err
	}
	if "@"+update.CallbackQuery.From.UserName != message.UserName {
		tg.TelegramBot.ShowAlert(update.CallbackQuery.ID, "Присутствие может отметить только преподаватель.")
		return errors.New("попытка отметить присутствие не преподавателем. Не совпадаение username " + "@" + update.CallbackQuery.From.UserName + " != " + message.UserName)
	}
	// Изменение сообщения для понимания работоспособности
	newText := removeLastLine(update.CallbackQuery.Message.Text) + "\n✅Вы отреагировали на сообщение. Спасибо!✅"
	msgToEdit := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, int(message.MsgID), newText)
	_, err = appBot.EditMessageLowPriority(msgToEdit)
	if err != nil {
		logger.Info("Не удалось изменить сообщение после успешной реакции пользователя: ", err)
		return fmt.Errorf("ошибка. Не удалось изменить сообщение")
	}
	logger.Info(update.CallbackQuery.Message.Text)
	// Обновление в основной БД
	updateRowConfig := UpdateRowConfig{
		SearchColumnName: "uid",
		SearchValue:      message.UID,
		NewColumnName:    "msg_is_pressed",
		NewValue:         true,
		Row:              model.Message{},
	}
	err = app.Message.UpdateRow(updateRowConfig)
	if err != nil {
		logger.Info("Не удалось обновить состояние сообщения: ", err)
		return err
	}

	// Удаление из временной БД callback
	err = app.CallBack.DeleteRecordByColumn("uid", update.CallbackQuery.Data, CallBack{})
	if err != nil {
		logger.Info("Не удалось удалить сообщение из бд CallBack: ", err)
	}
	return nil
}

// prepareLessonNotification форматирует структуру в текст, который будет отправлен пользователю вместе с кнопкой
func prepareLessonNotification(message model.Message) string {
	formattedTime := message.LessonTime.Format("2006-01-02 15:04")

	toTgString := fmt.Sprintf(`
	<strong>Урок будет в ближайшее время</strong>
	
	<strong>📋Название группы</strong>: %s
	<strong>⏰Время урока: %s</strong>

	<strong>🔢Номер группы:</strong> %d
	<strong>🙂Имя преподавателя: %s</strong>
	<strong>😊Ученики(активные):</strong> %d

	🆗Чтобы не пропустить урок, нажмите на кнопочку
	`, message.CourseName, formattedTime, message.CourseID, message.TeacherName, message.ActiveMember)
	return toTgString
}

func (appNotification *NotificationManager) AddHandlerButtonPress(message model.Message, msgID int) {
	// Вставляем сообщение для обработки нажатия кнопки
	callback := CallBack{
		UID:      message.UID,
		MsgID:    int64(msgID),
		ChatID:   message.ChatID,
		UserName: message.UserName,
	}
	appNotification.CallBack.InsertRow(&callback)
}

func removeLastLine(multilineString string) string {
	// Разбиваем строку на отдельные строки
	lines := strings.Split(multilineString, "\n")

	// Если количество строк меньше или равно 1, возвращаем оригинальную строку
	if len(lines) <= 1 {
		return multilineString
	}

	// Объединяем все строки, кроме последней
	return strings.Join(lines[:len(lines)-1], "\n")
}
