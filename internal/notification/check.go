package notification

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/model"
	"easycodeapp/internal/tg"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (app *NotificationManager) handleCheckMessages(results []model.Message) {
	for _, msg := range results {
		app.checkMessage(msg)
	}
}

// checkMessage обрабатывает партию сообщений
func (app *NotificationManager) checkMessage(message model.Message) {
	if !isSameYearMonthDay(time.Now(), message.LessonTime) {
		return
	}
	if !(hasDatePassed(message.MsgSendTime)) {
		return
	}

	// Если дата урока наступила
	if hasDatePassed(message.DelayTime) {
		// Ответ на сообщение был
		if message.MsgIsPressed {

			err := app.handleMsgWithReaction(message)
			if err != nil {
				logger.Info("Ошибка при обработке сообщения С реакцией: ", err)
				return
			}
			// не было ответа
		} else {
			// Если сообщение даже не было отправлено
			if !message.MsgIssent {
				err := app.handleMsgWithReaction(message)
				if err != nil {
					logger.Info("Ошибка при обработке сообщения С реакцией: ", err)
					return
				}
				// Если сообщение было отправлено
			} else {

				err := app.handleMsgWithoutReaction(message)
				if err != nil {
					logger.Info("Ошибка при обработке сообщения БЕЗ реакции: ", err)
					return
				}
			}
		}

		// Не дата урока наступила
	} else {
		// Ответ на сообщение был
		if message.MsgIsPressed {
			err := app.handleMsgWithReaction(message)
			if err != nil {
				logger.Info("Ошибка при обработке сообщения С реакцией: ", err)
				return
			}
			// не было ответа
		} else {
			return
		}
	}
}

// handleMsgWithoutReaction отправляет пользователю и в специальный чат сообщение о пропуске занятия, удаляет из БД сообщение
func (app *NotificationManager) handleMsgWithoutReaction(message model.Message) error {

	msgText := prepareMsgTextWithoutReaction(message)
	replyMsg := tgbotapi.NewMessage(config.File.TelegramConfig.NotificationChatId, msgText)
	replyMsg.ReplyToMessageID = config.File.TelegramConfig.AbsenteeismTopicID

	replyMsg.ParseMode = "html"
	_, err := tg.TelegramBot.SendMessageRepetLowPriority(replyMsg, 3)
	if err != nil {
		logger.Info("Не удалось отправить сообщение о удачной отправке")
	}

	err = app.Message.DeleteRecordByColumn("uid", message.UID, model.Message{})
	if err != nil {
		logger.Info("Не  удалось удалить сообщение при проверке старых сообщений: ", err)
	}

	// Изменение сообщения для понимания работоспособности
	newText := preapareLessonFailedNotification(message)

	msgToEdit := tgbotapi.NewEditMessageText(message.ChatID, int(message.MsgID), newText)
	msgToEdit.ParseMode = "html"
	_, err = tg.TelegramBot.EditMessageLowPriority(msgToEdit)
	if err != nil {
		logger.Info("Не удалось изменить сообщение для отметки отсутствия преподавателя: ", err)
	}

	err = app.Message.DeleteRecordByColumn("uid", message.UID, CallBack{})
	if err != nil {
		return err
	}

	return err
}

// prepareMsgTextWithoutReaction подготавливает отформатированный текст сообщения об отсутствии преподавателя
func prepareMsgTextWithoutReaction(message model.Message) string {

	formattedTime := message.LessonTime.Format("2006-01-02 15:04")
	text := fmt.Sprintf(`
	🚨Преподаватель не подтвердил присутствие на уроке.
	
	<strong>📋Название группы</strong>: %s
	<strong>⏰Время урока: %s</strong>
	<strong>🙂Имя преподавателя: %s</strong>
	`, message.CourseName, formattedTime, message.TeacherName)

	return text
}

// handleMsgWithReaction удаляет сообщение из БД
func (app *NotificationManager) handleMsgWithReaction(message model.Message) error {
	// Обновление в основной БД
	// Удаляем из основной бд и отправляем в старую
	tempMessage := OldMessage{message}
	err := app.Message.DeleteRecordByColumn("uid", message.UID, model.Message{})
	if err != nil {
		logger.Info("Не  удалось удалить сообщение при проверке старых сообщений: ", err)
	}
	err = app.Message.InsertRow(&tempMessage)
	if err != nil {
		logger.Info("Ошибка при вставке сообщения в таблицу старых сообщений: ", err)
	}

	return nil
}

func preapareLessonFailedNotification(message model.Message) string {
	formattedTime := message.LessonTime.Format("2006-01-02 15:04")

	toTgString := fmt.Sprintf(`
	<strong>Урок будет в ближайшее время</strong>
	
	<strong>📋Название группы</strong>: %s
	<strong>⏰Время урока: %s</strong>

	<strong>🔢Номер группы:</strong> %d
	<strong>🙂Имя преподавателя: %s</strong>
	<strong>😊Ученики(активные):</strong> %d

	❌Реакции на урок не было. С вами свяжется менеджер❌
	`, message.CourseName, formattedTime, message.CourseID, message.TeacherName, message.ActiveMember)
	return toTgString
}
