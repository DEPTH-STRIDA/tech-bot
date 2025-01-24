package tg

import (
	"easycodeapp/internal/cache"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	m "easycodeapp/internal/model"
	"easycodeapp/pkg/model"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

var Admin *AdminApp = &AdminApp{}

type AdminApp struct {
}

func InitAdminApp() error {
	Admin = &AdminApp{}

	go Admin.StartSendingMailing()
	go Admin.StartCheckingExpiredMailings()

	return nil
}

// CreateMailing создает новую рассылку в БД и возвращает её ID
func (app *AdminApp) CreateMailing(mailing m.Mailing) (int64, error) {
	logger.Info("Данные: ", mailing)
	result := db.DB.Create(&mailing)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create mailing: %w", result.Error)
	}

	return int64(mailing.ID), nil
}

// Обновленная функция StartCheckingExpiredMailings
func (app *AdminApp) StartCheckingExpiredMailings() {
	for {
		// Обновляем все просроченные рассылки за один запрос
		result := db.DB.Model(&m.Mailing{}).
			Where("created_at < ? AND mailing_expired = ?", time.Now().Add(-24*time.Hour), false).
			Update("mailing_expired", true)

		if result.Error != nil {
			logger.Error("Failed to update expired mailings", "error", result.Error)
		} else if result.RowsAffected > 0 {
			logger.Info("Expired mailings updated", "count", result.RowsAffected)

			// Обработка просроченных рассылок
			err := app.ProcessExpiredMailings()
			if err != nil {
				logger.Error("Failed to process expired mailings", "error", err)
			}
		} else {
			logger.Info("No expired mailings found, waiting for 30 minutes")
		}

		time.Sleep(30 * time.Minute)
	}
}

// ProcessExpiredMailings обрабатывает просроченные рассылки
func (app *AdminApp) ProcessExpiredMailings() error {
	logger.Info("ОЧИСТКА ПРОСРОЧЕННЫХ РАССЫЛОК")
	for {
		mailing, err := app.GetExpiredMailing() // Получаем одну просроченную рассылку
		if err != nil {
			if err.Error() == "record not found" {
				break // Если не найдено больше просроченных рассылок, выходим из цикла
			}
			return fmt.Errorf("failed to get expired mailing: %w", err)
		}

		// Проверяем необходимость кнопки
		if !mailing.Button {
			// Удаляем рассылку из БД
			err = DeleteMailing(mailing.ID)
			if err != nil {
				logger.Error("Failed to delete mailing", "mailingID", mailing.ID, "error", err)
			}
			continue
		}

		// Собираем пользователей, не отреагировавших на рассылку
		var nonReactedUsers []m.MailingStatus
		err = db.DB.Where("mailing_id = ? AND msg_is_reacted = false", mailing.ID).Find(&nonReactedUsers).Error
		if err != nil {
			logger.Error("Failed to get non-reacted users", "mailingID", mailing.ID, "error", err)
			continue
		}

		// Формируем сообщение для администратора
		msgText := fmt.Sprintf("<strong>Рассылка просрочена</strong>\n<strong>Когорта:</strong> %s\n<strong>Тип рассылки:</strong> %s\n", mailing.CohortName, mailing.MailingType)
		if len(nonReactedUsers) > 0 {
			msgText += "Пользователи, не отреагировавшие на рассылку:\n"
			for _, user := range nonReactedUsers {
				msgText += fmt.Sprintf("%s\n", user.UserName)
			}
		} else {
			msgText += "<strong>Все пользователи отреагировали на рассылку.</strong>\n"
		}

		// Разбиваем сообщение на части, если оно превышает 4096 символов
		var messages []string
		currentMessage := msgText

		for len(currentMessage) > 0 {
			if len(currentMessage) > 4096 {
				// Находим последний перенос строки, чтобы не обрезать на середине
				lastNewline := strings.LastIndex(currentMessage[:4096], "\n")
				if lastNewline == -1 {
					lastNewline = 4096 // Если перенос не найден, обрезаем на 4096
				}
				messages = append(messages, currentMessage[:lastNewline])
				currentMessage = currentMessage[lastNewline:] // Оставшаяся часть сообщения
			} else {
				messages = append(messages, currentMessage)
				break
			}
		}

		// Отправляем сообщения админу
		for _, adminMsgText := range messages {
			adminMsg := tgbotapi.NewMessage(mailing.AuthorTgID, adminMsgText)
			adminMsg.ParseMode = "html"
			TelegramBot.SendMessage(adminMsg)
		}

		// Удаляем рассылку из БД
		err = DeleteMailing(mailing.ID)
		if err != nil {
			logger.Error("Failed to delete mailing", "mailingID", mailing.ID, "error", err)
		}
	}
	return nil
}

// GetExpiredMailing получает одну просроченную рассылку
func (app *AdminApp) GetExpiredMailing() (m.Mailing, error) {
	var mailing m.Mailing
	if err := db.DB.Where("mailing_expired = true").First(&mailing).Error; err != nil {
		return m.Mailing{}, err
	}
	return mailing, nil
}

// DeleteMailing удаляет рассылку по ID
func DeleteMailing(mailingID uint) error {
	// Получаем все статусы рассылки
	var statuses []m.MailingStatus
	if err := db.DB.Where("mailing_id = ?", mailingID).Find(&statuses).Error; err != nil {
		return err
	}

	for i := 0; i < len(statuses); i++ {
		msgToDelete := tgbotapi.NewDeleteMessage(statuses[i].TgID, statuses[i].MsgID)
		TelegramBot.SendDeleteMessage(msgToDelete)
	}

	if err := db.DB.Delete(&m.Mailing{}, mailingID).Error; err != nil {
		return err
	}
	return nil
}

// Обновленная функция StartSendingMailing
func (app *AdminApp) StartSendingMailing() {
	for {
		mailing, err := app.GetActiveMailing()
		if err != nil {
			if err.Error() == "active mailing not found" {
				logger.Info("No active mailings found, finishing")
				time.Sleep(1 * time.Minute)
				continue
			}
			logger.Error("Failed to get active mailing", "error", err)
			TelegramBot.SendAllAdmins("(ПАУЗА 5 минут)Failed to get active mailing error" + err.Error())
			time.Sleep(5 * time.Minute)
			continue
		}

		for _, status := range mailing.MailingStatuses {
			if status.MsgIsSent {
				continue
			}

			msgText := mailing.MessageText
			msg := tgbotapi.NewMessage(status.TgID, msgText)
			msg.ParseMode = "html"
			// msg.Entities = mailing.Entities // Добавляем сохраненные сущности

			if mailing.Button {
				msgText += "\n\nЧтобы подтвердить получение нажмите кнопку ⬇️"
				msg.Text = msgText
				msg.ReplyMarkup = CreateInlineKeyboard([][]ButtonData{{ButtonData{
					Text: "Принято! ✔️",
					Data: fmt.Sprintf(`{"ActionType":"Mailing","MailingID":%d,"StatusID":%d}`, mailing.ID, status.ID),
				}}})
			}

			var update map[string]interface{}
			chatID, sendedMsg, err := TelegramBot.SendMessageUnkownChatIdD(msg)
			if err != nil {
				update = map[string]interface{}{
					"msg_is_sent": false,
					"send_failed": true,
				}
			} else {
				logger.Info("Сообщение отправлено", " msgID ", sendedMsg.MessageID)
				update = map[string]interface{}{
					"msg_is_sent": true,
					"send_failed": false,
					"msg_id":      sendedMsg.MessageID,
					"tg_id":       chatID,
				}
			}

			err = app.UpdateMailingStatus(status.ID, update)
			if err != nil {
				logger.Error("Failed to update mailing status",
					"statusID", status.ID,
					"chatID", chatID,
					"error", err,
					"update", update)
			}
		}

		err = Admin.UpdateMailing(mailing.ID, map[string]interface{}{"mailing_finished": true})
		if err != nil {
			logger.Error(err)
		} else {
			// Формируем сообщение с результатами
			msgText := "<strong>Рассылка отправлена</strong>\n"
			msgText += "<strong>ID: " + fmt.Sprint(mailing.ID) + "</strong>\n"
			if mailing.MailingType != MailingTeamChat.String() {
				msgText += "<strong>Когорта:</strong>  " + mailing.CohortName + "\n"
			}
			msgText += "<strong>Тип рассылки:</strong>  " + mailing.MailingType + "\n"

			// baseLength := len(msgText)

			if mailing.MessageText == "" {
				msgText += "\n<strong>Текст сообщения пуст</strong>\n\n"
			} else {
				msgText += "<strong>Текст сообщения:</strong> \n\n" + mailing.MessageText + "\n\n"
			}
			msgText += "Получить полную статистику⬇️"

			msg := tgbotapi.NewMessage(mailing.AuthorTgID, msgText)

			// // Если есть сущности, сдвигаем их на длину префикса
			// if len(mailing.Entities) > 0 {
			// 	// shiftedEntities := make([]tgbotapi.MessageEntity, len(mailing.Entities))
			// 	for i, entity := range mailing.Entities {
			// 		shiftedEntities[i] = entity
			// 		shiftedEntities[i].Offset += baseLength + len("<strong>Текст сообщения:</strong> \n\n")
			// 	}
			// 	msg.Entities = shiftedEntities
			// }

			msg.ParseMode = "html"
			msg.ReplyMarkup = CreateInlineKeyboard([][]ButtonData{{
				ButtonData{
					Text: "Полная статистика",
					Data: fmt.Sprintf(`{"ActionType":"Statistic","MailingID":%d}`, mailing.ID),
				},
				ButtonData{
					Text: "Отменить рассылку",
					Data: fmt.Sprintf(`{"ActionType":"StatisticDeleting","MailingID":%d}`, mailing.ID),
				},
			}})

			TelegramBot.SendMessageLowPriority(msg)
		}
	}
}

func (app *AdminApp) UpdateMailing(mailingID uint, update map[string]interface{}) error {
	result := db.DB.Model(&m.Mailing{}).
		Where("id = ?", mailingID).
		Updates(update)

	if result.Error != nil {
		return fmt.Errorf("failed to update mailing status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("mailing status with ID %d not found", mailingID)
	}

	return nil
}

func (app *AdminApp) UpdateMailingStatus(statusID uint, update map[string]interface{}) error {
	result := db.DB.Model(&m.MailingStatus{}).
		Where("id = ?", statusID).
		Updates(update)

	if result.Error != nil {
		return fmt.Errorf("failed to update mailing status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("mailing status with ID %d not found", statusID)
	}

	return nil
}

// MarkMailingFinished помечает рассылку как завершенную
func (app *AdminApp) MarkMailingFinished(mailingID uint) error {
	result := db.DB.Model(&m.Mailing{}).
		Where("id = ?", mailingID).
		Update("mailing_finished", true)

	if result.Error != nil {
		return fmt.Errorf("failed to mark mailing as finished: %w", result.Error)
	}

	return nil
}

// GetActiveMailing получает активную рассылку вместе со статусами пользователей
// Активная рассылка - это не удаленная, не завершенная и не просроченная рассылка
func (app *AdminApp) GetActiveMailing() (*m.Mailing, error) {
	var mailing m.Mailing

	result := db.DB.Preload("MailingStatuses").
		Where("mailing_finished = ?", false).
		Where("mailing_expired = ?", false).
		First(&mailing)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("active mailing not found")
		}
		return nil, fmt.Errorf("failed to get mailing: %w", result.Error)
	}

	return &mailing, nil
}

// UpdateMailing обновляет поля в главной таблице
func UpdateMailing(db *gorm.DB, mailingID uint) error {
	// Обновление нескольких полей в рассылке
	result := db.Model(&m.Mailing{}).
		Where("id = ?", mailingID).
		Updates(map[string]interface{}{
			"message_text":     "Обновленный текст сообщения",
			"mailing_finished": true,
			"mailing_expired":  true,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update mailing: %w", result.Error)
	}

	fmt.Printf("Updated mailing. Rows affected: %d\n", result.RowsAffected)
	return nil
}

// GetMailingWithStatuses загружает рассылку вместе со всеми статусами
func GetMailingWithStatuses(db *gorm.DB, mailingID uint) (*m.Mailing, error) {
	var mailing m.Mailing
	result := db.Preload("MailingStatuses").First(&mailing, mailingID)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get mailing: %w", result.Error)
	}

	return &mailing, nil
}

// GetMailingWithStatuses получает рассылку и её статусы по переданному ID
func (app *AdminApp) GetMailingWithStatuses(mailingID int64) (m.Mailing, []m.MailingStatus, error) {
	var mailing m.Mailing
	var statuses []m.MailingStatus

	// Получаем рассылку по ID
	if err := db.DB.First(&mailing, mailingID).Error; err != nil {
		return m.Mailing{}, nil, fmt.Errorf("failed to get mailing: %w", err)
	}

	// Получаем статусы рассылки
	if err := db.DB.Where("mailing_id = ?", mailingID).Find(&statuses).Error; err != nil {
		return m.Mailing{}, nil, fmt.Errorf("failed to get mailing statuses: %w", err)
	}

	return mailing, statuses, nil
}

func (app *AdminApp) UpdateStatusReaction(statusID int64) (*m.MailingStatus, error) {
	var status m.MailingStatus

	// Получаем статус по ID
	if err := db.DB.First(&status, statusID).Error; err != nil {
		return nil, fmt.Errorf("failed to find status: %w", err)
	}

	// Обновляем поле MsgIsReacted
	status.MsgIsReacted = true

	// Сохраняем изменения в базе данных
	if err := db.DB.Save(&status).Error; err != nil {
		return nil, fmt.Errorf("failed to update status reaction: %w", err)
	}

	return &status, nil
}

// GetAllMailings получает все неудаленные рассылки из базы данных
func (app *AdminApp) GetAllMailings() ([]m.Mailing, error) {
	var mailings []m.Mailing

	// Получаем все рассылки, включая связанные статусы
	result := db.DB.
		Preload("MailingStatuses").
		Find(&mailings)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get mailings: %w", result.Error)
	}

	// Если рассылок нет, возвращаем пустой слайс
	if len(mailings) == 0 {
		return []m.Mailing{}, nil
	}

	return mailings, nil
}

// GetMailingsByFilter получает рассылки с фильтрацией по разным параметрам
func (app *AdminApp) GetMailingsByFilter(filters map[string]interface{}) ([]m.Mailing, error) {
	var mailings []m.Mailing

	query := db.DB.Preload("MailingStatuses")

	// Применяем фильтры, если они есть
	for field, value := range filters {
		query = query.Where(field+" = ?", value)
	}

	// Выполняем запрос
	result := query.Find(&mailings)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get mailings with filters: %w", result.Error)
	}

	return mailings, nil
}
func createMailingStatuses(user CachedUser) ([]m.MailingStatus, error) {
	mailingStatuses := make([]m.MailingStatus, 0)

	// Логируем информацию о пользователе и его типе рассылки
	logger.Info(fmt.Sprintf("Создание статусов рассылки для пользователя: %s, тип рассылки: %s", user.CohortsName, user.MailingType.String()))

	switch user.MailingType {
	case MailingPrivateMessage:
		// Когорта
		cohort := cache.TelegramCacheApp.GetCohortByName(user.CohortsName)
		logger.Info(fmt.Sprintf("Получена когорта: %v", cohort))

		// Данные преподов
		var users []model.User
		result := db.DB.Find(&users)
		if result.Error != nil {
			logger.Error("Ошибка при получении данных пользователей: " + result.Error.Error())
			return nil, result.Error
		}
		logger.Info(fmt.Sprintf("Найдено пользователей: %d", len(users)))

		// Обход препода из когорты
		for _, v := range cohort {
			cleanedCohortName := strings.ToLower(strings.TrimSpace(v)) // Удаляем пробелы и приводим к нижнему регистру
			// logger.Info(fmt.Sprintf("Обрабтка пользователя из когорты: %s", cleanedCohortName)) // Логируем текущего пользователя из когорты
			for _, value := range users {
				cleanedUserName := strings.ToLower(strings.TrimSpace(value.TeacherName)) // Удаляем пробелы и приводим к нижнему регистру
				// logger.Info(fmt.Sprintf("Проверка пользователя: %s", cleanedUserName))  // Логируем проверяемого пользователя
				// Если найдено совпадение, т добавляем в рассылку
				if cleanedCohortName == cleanedUserName {
					mailingStatuses = append(mailingStatuses, m.MailingStatus{
						UserName: value.UserName,
						TgID:     value.UserID,
					})
					// logger.Info(fmt.Sprintf("Добавлен статус рассылки для пользователя: %s, TgID: %d", value.UserName, value.UserID))
				}
			}
		}

	case MailingManagerChat:
		logger.Info("Обработка рассылки в чат менеджеров")
		cohort := cache.TelegramCacheApp.GetCohortByName(user.CohortsName)
		logger.Info(fmt.Sprintf("Получена когорта: %v", cohort))

		// Данные преподов
		var users []model.User
		result := db.DB.Find(&users)
		if result.Error != nil {
			logger.Error("Ошибка при получении данных пользователей: " + result.Error.Error())
			return nil, result.Error
		}
		logger.Info(fmt.Sprintf("Найдено пользователей: %d", len(users)))

		for _, v := range cohort {
			cleanedCohortName := strings.ToLower(strings.TrimSpace(v))                           // Удаляем пробелы и приводим к нижнему регистру
			logger.Info(fmt.Sprintf("Обработка пользователя из когорты: %s", cleanedCohortName)) // Логируем текущего пользователя из когорты
			for _, value := range users {
				cleanedUserName := strings.ToLower(strings.TrimSpace(value.TeacherName)) // Удаляем пробелы и приводим к нижнему регистру
				logger.Info(fmt.Sprintf("Проверка пользователя: %s", cleanedUserName))   // Логируем проверяемого пользователя
				// Если найдено совпадение, то добавляем в рассылку
				if cleanedCohortName == cleanedUserName {
					mailingStatuses = append(mailingStatuses, m.MailingStatus{
						UserName: "Чат с менеджерами " + value.TeacherName,
						TgID:     value.ChatID,
					})
					logger.Info(fmt.Sprintf("Добавлен статус рассылки для чата менеджера: %s, TgID: %d", value.TeacherName, value.ChatID))
				}
			}
		}

	case MailingTeamChat:
		logger.Info("Обработка рассылки в командный чат")
		chats := cache.TelegramCacheApp.GetTeamChats()
		for _, v := range chats {
			logger.Info(fmt.Sprintf("Добавление командного чата с TgID: %d", v)) // Логируем добавление командного чата
			mailingStatuses = append(mailingStatuses, m.MailingStatus{
				UserName: "Командный чат: " + strconv.FormatInt(v, 10),
				TgID:     v,
			})
			logger.Info(fmt.Sprintf("Добавлен статус рассылки для командного чата: TgID: %d", v))
		}
	}

	logger.Info(fmt.Sprintf("Создано статусов рассылки: %d", len(mailingStatuses)))
	return mailingStatuses, nil
}
