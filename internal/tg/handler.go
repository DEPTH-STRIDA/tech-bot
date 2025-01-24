package tg

import (
	"easycodeapp/internal/cache"
	"easycodeapp/internal/config"
	"easycodeapp/internal/googlesheet"
	"easycodeapp/internal/infrastructure/clickhouse"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	m "easycodeapp/internal/model"
	"easycodeapp/internal/utils"
	"easycodeapp/pkg/model"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

const (
	GlobalState           = "GlobalState"
	MenuStateName         = "MenuStateName"
	ChoosingMailingType   = "ChoosingMailingType"
	ChoosingCohortsType   = "ChoosingCohortsType"
	EnterMailingType      = "EnterMailingType"
	ChoosingMailingToStat = "ChoosingMailingToStat"
)

type MailingType int

const (
	MailingPrivateMessage MailingType = iota // 0
	MailingManagerChat                       // 1
	MailingTeamChat                          // 2
)

func (m MailingType) String() string {
	switch m {
	case MailingPrivateMessage:
		return "Личные сообщения"
	case MailingManagerChat:
		return "Чат с менеджерами"
	case MailingTeamChat:
		return "Командный чат"
	default:
		return ""
	}
}

func UpdateTelegramID(app *Bot, userName string, telegramID int64) error {
	// Проверяем существование пользователя
	var user model.User
	result := db.DB.Where("user_name = ?", "@"+userName).First(&user)

	// Если пользователь не найден, возвращаемся без ошибки
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	// Если произошла другая ошибка при поиске
	if result.Error != nil {
		return errors.New("database error: " + result.Error.Error())
	}

	// Обновляем только UserID
	if err := db.DB.Model(&user).Update("user_id", telegramID).Error; err != nil {
		return errors.New("failed to update telegram id: " + err.Error())
	}

	return nil
}
func IsAdminSilent(b *Bot, update tgbotapi.Update) error {
	isAdmin := cache.TelegramCacheApp.IsAdmin(update.SentFrom().ID)

	if !isAdmin {
		return fmt.Errorf("ользователь не админ")
	}
	return nil
}

func IsAdmin(b *Bot, update tgbotapi.Update) error {
	isAdmin := cache.TelegramCacheApp.IsAdmin(update.SentFrom().ID)

	if !isAdmin {
		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Проверка не прошла ❌\nВы не являетесь админом. Запросите админку у @slice13")
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		b.SendMessage(msg)
		return fmt.Errorf("ользователь не админ")
	}
	return nil
}

// Обработчик для получения статистики заявок
var HandleStat = Handler{
	Func: func(app *Bot, update tgbotapi.Update) error {
		var forms []model.Form

		// Получаем текущую дату
		today := time.Now().Truncate(24 * time.Hour) // Убираем время, оставляя только дату
		tomorrow := today.Add(24 * time.Hour)        // Получаем завтрашнюю дату

		// Запрос на получение форм, созданных сегодня, отсортированных по дате создания
		err := db.DB.Where("created_at >= ? AND created_at < ?", today, tomorrow).
			Order("created_at ASC"). // Добавляем сортировку по дате создания
			Find(&forms).Error

		// Формируем текст для вывода
		var resultText strings.Builder
		resultText.WriteString("Сообщение\nЧат замен | Спецназ | Строка в таблице\n") // Заголовок

		replaceChatIDPrefix := config.File.TelegramConfig.ReplaceChatID
		replaceChatIDstr := fmt.Sprintf("%d", replaceChatIDPrefix)
		replaceChatID := strings.TrimPrefix(replaceChatIDstr, "-100")

		emergencyChatIDPrefix := config.File.TelegramConfig.EmergencyChatID
		emergencyChatIDstr := fmt.Sprintf("%d", emergencyChatIDPrefix)
		emergencyChatID := strings.TrimPrefix(emergencyChatIDstr, "-100")

		for i, form := range forms {
			resultText.WriteString(fmt.Sprintf("%d) ", i+1))

			// Обработка Telegram ID
			if form.ReplaceTgStatus {
				resultText.WriteString(fmt.Sprintf("https://t.me/c/%s/%d | ", replaceChatID, form.ReplaceMsgId))
			} else {
				resultText.WriteString("none | ")
			}

			// Обработка Emergency Telegram ID
			if form.EmergencyTgStatus {
				resultText.WriteString(fmt.Sprintf("https://t.me/c/%s/%d | ", emergencyChatID, form.EmergencyMsgId))
			} else {
				resultText.WriteString("none | ")
			}

			// Обработка Google Sheets
			if form.GoogleSheetStatus {
				resultText.WriteString(fmt.Sprintf("%d\n", form.GoogleSheetLineNumber))
			} else {
				resultText.WriteString("none\n")
			}

			resultText.WriteString("\n") // Добавляем пустую строку между заявками
		}

		// Отправка сообщения с результатами, разбивая на части, если необходимо
		messageText := resultText.String()
		maxMessageLength := 4000

		for len(messageText) > 0 {
			// Если длина сообщения больше максимальной, ищем точку разбиения
			if len(messageText) > maxMessageLength {
				// Находим последний перенос строки или пробел перед пределом
				breakPoint := strings.LastIndex(messageText[:maxMessageLength], "\n")
				if breakPoint == -1 {
					breakPoint = strings.LastIndex(messageText[:maxMessageLength], " ")
				}
				if breakPoint == -1 {
					breakPoint = maxMessageLength // Если нет пробела или переноса, просто обрезаем
				}

				// Отправляем часть сообщения
				msg := tgbotapi.NewMessage(update.SentFrom().ID, messageText[:breakPoint])
				msg.ParseMode = "html"
				_, err = app.SendMessage(msg)
				if err != nil {
					logger.Error("Не удалось отправить сообщение с заявками", "Error", err)
					return err
				}

				// Удаляем отправленную часть из сообщения
				messageText = messageText[breakPoint:]
			} else {
				// Если длина сообщения меньше максимальной, отправляем его целиком
				msg := tgbotapi.NewMessage(update.SentFrom().ID, messageText)
				msg.ParseMode = "html"
				_, err = app.SendMessage(msg)
				if err != nil {
					logger.Error("Не удалось отправить сообщение с заявками", "Error", err)
					return err
				}
				break // Выходим из цикла
			}
		}

		return nil
	},
	Description: "Присылает пподробную информацию по заявка, отправленным сегодня.\nСсылка на форму в чате замен, спецназа и номер строки в таблице.",
}

// Обработчик для получения статистики заявок с возможностью указания даты
var HandleStatDate = Handler{
	Func: func(app *Bot, update tgbotapi.Update) error {
		err := IsAdminSilent(app, update)
		if err != nil {
			return err
		}

		var forms []model.Form

		// Получаем текст сообщения
		messageText := update.Message.Text

		// Инициализируем переменную для даты
		var dateToCheck time.Time
		err = nil
		// Проверяем наличие команды /forms с датой
		if strings.HasPrefix(messageText, "/forms ") {
			dateString := strings.TrimPrefix(messageText, "/forms ")
			dateToCheck, err = time.Parse("02.01.2006", dateString) // Парсим дату в формате "дд.мм.гггг"
			if err != nil {
				return fmt.Errorf("неверный формат даты: %s", dateString)
			}
		} else {
			return fmt.Errorf("неверный формат команды")
		}

		// Получаем начало и конец дня для выбранной даты
		today := dateToCheck
		tomorrow := today.Add(24 * time.Hour)

		// Запрос на получение форм, созданных за указанную дату, отсортированных по дате создания
		err = db.DB.Where("created_at >= ? AND created_at < ?", today, tomorrow).
			Order("created_at ASC"). // Добавляем сортировку по дате создания
			Find(&forms).Error

		if err != nil {
			logger.Error("Ошибка при получении заявок из базы данных", "Error", err)
			return err
		}

		// Формируем текст для вывода
		var resultText strings.Builder
		resultText.WriteString(fmt.Sprintf("Сообщение за дату: %s\nЧат замен | Спецназ | Строка в таблице\n", today.Format("02.01.2006"))) // Заголовок

		replaceChatIDPrefix := config.File.TelegramConfig.ReplaceChatID
		replaceChatIDstr := fmt.Sprintf("%d", replaceChatIDPrefix)
		replaceChatID := strings.TrimPrefix(replaceChatIDstr, "-100")

		emergencyChatIDPrefix := config.File.TelegramConfig.EmergencyChatID
		emergencyChatIDstr := fmt.Sprintf("%d", emergencyChatIDPrefix)
		emergencyChatID := strings.TrimPrefix(emergencyChatIDstr, "-100")

		for i, form := range forms {
			resultText.WriteString(fmt.Sprintf("%d) ", i+1))

			// Обработка Telegram ID
			if form.ReplaceTgStatus {
				resultText.WriteString(fmt.Sprintf("https://t.me/c/%s/%d | ", replaceChatID, form.ReplaceMsgId))
			} else {
				resultText.WriteString("none | ")
			}

			// Обработка Emergency Telegram ID
			if form.EmergencyTgStatus {
				resultText.WriteString(fmt.Sprintf("https://t.me/c/%s/%d | ", emergencyChatID, form.EmergencyMsgId))
			} else {
				resultText.WriteString("none | ")
			}

			// Обработка Google Sheets
			if form.GoogleSheetStatus {
				resultText.WriteString(fmt.Sprintf("%d\n", form.GoogleSheetLineNumber))
			} else {
				resultText.WriteString("none\n")
			}

			resultText.WriteString("\n") // Добавляем пустую строку между заявками
		}

		// Отправка сообщения с результатами, разбивая на части, если необходимо
		messageText = resultText.String()
		maxMessageLength := 4000

		for len(messageText) > 0 {
			// Если длина сообщения больше максимальной, ищем точку разбиения
			if len(messageText) > maxMessageLength {
				// Находим последний перенос строки или пробел перед пределом
				breakPoint := strings.LastIndex(messageText[:maxMessageLength], "\n")
				if breakPoint == -1 {
					breakPoint = strings.LastIndex(messageText[:maxMessageLength], " ")
				}
				if breakPoint == -1 {
					breakPoint = maxMessageLength // Если нет пробела или переноса, просто обрезаем
				}

				// Отправляем часть сообщения
				msg := tgbotapi.NewMessage(update.SentFrom().ID, messageText[:breakPoint])
				msg.ParseMode = "html"
				_, err = app.SendMessage(msg)
				if err != nil {
					logger.Error("Не удалось отправить сообщение с заявками", "Error", err)
					return err
				}

				// Удаляем отправленную часть из сообщения
				messageText = messageText[breakPoint:]
			} else {
				// Если длина сообщения меньше максимальной, отправляем его целиком
				msg := tgbotapi.NewMessage(update.SentFrom().ID, messageText)
				msg.ParseMode = "html"
				_, err = app.SendMessage(msg)
				if err != nil {
					logger.Error("Не удалось отправить сообщение с заявками", "Error", err)
					return err
				}
				break // Выходим из цикла
			}
		}

		return nil
	},
	Description: "Присылает подробную информацию по заявкам, отправленным за указанную дату или за текущий день.",
}

// Обработчик для получения уроков пользователя
var HandleGetMyLessons = Handler{
	Func: func(app *Bot, update tgbotapi.Update) error {
		logger.Info("Начало обработки запроса HandleGetMyLessons",
			"UserID", update.SentFrom().ID,
			"UserName", update.SentFrom().FirstName+" "+update.SentFrom().LastName)

		notFondMsg := tgbotapi.NewMessage(update.SentFrom().ID,
			update.SentFrom().FirstName+" "+update.SentFrom().LastName+" ,уроки не найдены. Пожалуйста, проверьте данные в CRM")

		// Получение пользователя, для получения CRM_ID
		var user model.User
		logger.Info("Запрос пользователя из базы данных", "UserID", update.SentFrom().ID)
		err := db.DB.Where("user_name = ?", "@"+update.SentFrom().UserName).First(&user).Error
		if err != nil {
			logger.Error("Пользователь не найден в базе данных", "Error", err)
			app.SendMessage(notFondMsg)
			return err
		}
		logger.Info("Пользователь успешно получен", "User", user)

		// Получение уроков по CRM_ID
		var messages []m.CachedMessage
		logger.Info("Запрос уроков из базы данных", "CRMID", user.CRMID)

		// Получаем текущую дату
		today := time.Now().Truncate(24 * time.Hour) // Убираем время, оставляя только дату

		// Запрос на получение уроков за сегодня
		err = db.DB.Where("crm_id = ? AND DATE(lesson_time) = ?", user.CRMID, today).Order("lesson_time asc").Find(&messages).Error
		if err != nil {
			logger.Error("Ошибка при получении уроков из базы данных", "Error", err)
			app.SendMessage(notFondMsg)
			return err
		}
		logger.Info("Уроки успешно получены", "КоличествоУроков", len(messages))

		// Если ничего не найдено, без ошибки
		if len(messages) == 0 || messages == nil {
			logger.Info("Уроки не найдены для пользователя", "CRMID", user.CRMID)
			app.SendMessage(notFondMsg)
			return nil
		}

		text := fmt.Sprintf("<strong>Имя преподавателя: %s</strong>\n\n", messages[0].TeacherName)
		for _, msg := range messages {
			text += fmt.Sprintf(`
			<strong>📋Название группы</strong>: %s
			<strong>⏰Время урока:</strong> %s

			`,
				msg.CourseName, msg.LessonTime.Format("2006-01-02 15:04"))
		}
		logger.Info("Сформирован текст сообщения с уроками", "Текст", text)

		msg := tgbotapi.NewMessage(update.FromChat().ID, text)
		msg.ParseMode = "html"
		logger.Info("Отправка сообщения с уоками", "ChatID", update.FromChat().ID)
		_, err = app.SendMessage(msg)
		if err != nil {
			logger.Error("Не удалось отправить сообщение с уроками", "Error", err)
			return err
		}

		logger.Info("Сообщение с уроками успешно отправлено", "ChatID", update.FromChat().ID)
		return nil
	},
	Description: "Отправляет список уроков преподавателя за сегодня.",
}

// Обработчик для команды /start
var HandleStartMessage = Handler{
	Func: func(app *Bot, update tgbotapi.Update) error {
		UpdateTelegramID(app, update.SentFrom().UserName, update.SentFrom().ID)

		conf := config.File.TelegramConfig
		chatID := update.Message.Chat.ID

		// Сообщение со стикером
		_, err := app.SendSticker(conf.StartStickerID, chatID)
		if err != nil {
			return utils.HandleError(err)
		}

		// Отправка нового сообщения
		_, err = app.SendMessage(tgbotapi.NewMessage(update.Message.Chat.ID, conf.StartMsg))
		if err != nil {
			return utils.HandleError(err)
		}

		// Открепление всех сообщений
		_, err = app.SendUnPinAllMessageEvent(update.Message.From.UserName, chatID)
		if err != nil {
			return utils.HandleError(err)
		}

		// Отправка сообщения с текстом для закрепления
		pinupMsg, err := app.SendMessage(tgbotapi.NewMessage(chatID, conf.PinUpMsg))
		if err != nil {
			return utils.HandleError(err)
		}

		// Закрепление сообщения
		_, err = app.SendPinMessageEvent(pinupMsg.MessageID, chatID, true)
		if err != nil {
			return utils.HandleError(err)
		}

		return nil
	},
	Description: "Присылает приветственные сообщения и закрепляет одно сообщение с инструкцией.",
}

// Обработчик для получения статистики за сегодня
var HandleGetTodayStatistics = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		if update.Message.Chat.ID != config.File.TelegramConfig.BotTgChat {
			return nil
		}

		if clickhouse.ClickHouseApp == nil {
			_, err := b.SendMessage(tgbotapi.NewMessage(update.Message.From.ID, "Пожалуйста, подождите. Приложение еще загружается."))
			if err != nil {
				utils.HandleError(err)
			}

		}

		now := time.Now()
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

		forms, transfer, urgentForms, err := db.GetFormsByDateRange(startDate, endDate)
		if err != nil {
			fmt.Println(err)
			return err
		}

		textMsg := fmt.Sprint("Статистика за ", startDate.Format("2006-01-02")) + fmt.Sprintf("\n\nКоличество замен: %d\n", forms)
		textMsg += fmt.Sprintf("Количество срочных замен: %d\n", urgentForms)
		textMsg += fmt.Sprintf("Количество переносов: %d\n", transfer)

		textMsg += "\n/tdstat - получить статистику за сегодня"
		textMsg += "\n/ydstat - получить статистику за вчера"

		newMsg := tgbotapi.NewMessage(config.File.TelegramConfig.BotTgChat, textMsg)
		newMsg.ParseMode = "html"
		newMsg.ReplyToMessageID = config.File.TelegramConfig.StatTopicId

		_, err = b.SendMessage(newMsg)
		if err != nil {
			utils.HandleError(err)
		}

		return nil
	},
	Description: "Присылает статистику замен/переносов за сегодня. Пришлет сообщение в чат \"Бот тг\" в топик \"Статистика\". Только общая информация: кол-во и тип заявокк.",
}

var HandleGetYesterdayStatistics = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		if update.Message.Chat.ID != config.File.TelegramConfig.BotTgChat {
			return nil
		}

		if clickhouse.ClickHouseApp == nil {
			_, err := b.SendMessage(tgbotapi.NewMessage(update.Message.From.ID, "Пожалуйста, подождите. Приложение еще загружается."))
			if err != nil {
				utils.HandleError(err)
			}
		}

		now := time.Now()
		startDate := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		forms, transfer, urgentForms, err := db.GetFormsByDateRange(startDate, endDate)
		if err != nil {
			fmt.Println(err)
			return err
		}

		textMsg := fmt.Sprint("Статистика за ", startDate.Format("2006-01-02")) + fmt.Sprintf("\n\nКоличество замен: %d\n", forms)
		textMsg += fmt.Sprintf("Количество срочных замен: %d\n", transfer)
		textMsg += fmt.Sprintf("Количество переносов: %d\n", urgentForms)

		textMsg += "\n/tdstat - получить статистику за сегодня"
		textMsg += "\n/ydstat - получить статистику за вчера"

		newMsg := tgbotapi.NewMessage(config.File.TelegramConfig.BotTgChat, textMsg)
		newMsg.ParseMode = "html"
		newMsg.ReplyToMessageID = config.File.TelegramConfig.StatTopicId

		_, err = b.SendMessage(newMsg)
		if err != nil {
			utils.HandleError(err)
		}

		return nil
	},
	Description: "Присылает статистику замен/переносов за вчера. Пришлет сообщение в чат \"Бот тг\" в топик \"Статистика\". Только общая информация: кол-во и тип заявокк.",
}

// Обработчик для меню
var HandleMenu = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		b.SetUserState(update.SentFrom().ID, MenuStateName)
		b.HandleUserState(update, b.states[MenuStateName])

		buttons := []string{"Новая рассылка",
			"Активные рассылки",
			"Обновить данные",
			"ДР препопадавателей",
			"Перезагрузить бота",
			"Закрыть",
		}
		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Меню админа открыто")
		msg.ReplyMarkup = CreateKeyboard(buttons, 2)
		b.SendMessage(msg)

		user := b.sessions.Get(update.SentFrom().ID)

		user.MailingMessagetext = ""
		user.CohortsName = ""
		user.MailingType = -1
		b.sessions.Set(update.SentFrom().ID, user)

		return nil
	},
	Description: "Открывает меню админа с доступными действиями.",
}

func ActionOnMailingType(mailingType MailingType) Handler {
	return Handler{
		Func: func(b *Bot, update tgbotapi.Update) error {
			err := IsAdmin(b, update)
			if err != nil {
				return err
			}

			userID := update.SentFrom().ID
			user := b.sessions.Get(userID)
			user.MailingType = mailingType
			b.sessions.Set(userID, user)

			if user.MailingType == MailingTeamChat {
				b.SetUserState(update.SentFrom().ID, EnterMailingType)
				msg := tgbotapi.NewMessage(update.SentFrom().ID, "Введите  текст сообщения")
				msg.ReplyMarkup = CreateKeyboard([]string{"Закрыть", "Назад"}, 2)
				b.SendMessage(msg)
			} else {
				b.SetUserState(update.SentFrom().ID, ChoosingCohortsType)
				cohorts := append(cache.TelegramCacheApp.GetCohortsNames(), "Назад", "Закрыть")
				msg := tgbotapi.NewMessage(update.SentFrom().ID, "Выберите кагорту для рассылки")
				msg.ReplyMarkup = CreateKeyboard(cohorts, 4)
				b.SendMessage(msg)
			}

			return nil
		},
		Description: "",
	}
}

func formatReadyMailMsg(user CachedUser) string {
	msg := "<strong>Подготовлена рассылка</strong>\n"
	if user.MailingType != MailingTeamChat {
		msg += "<strong>Когорта:</strong>  " + user.CohortsName + "\n"
	}
	msg += "<strong>Тип рассылки:</strong>  " + user.MailingType.String() + "\n"
	if user.MailingMessagetext == "" {
		msg += "\n<strong>Текст сообщения пуст</strong>\n " + "\nЕсли в тексте ошибка, то пришлите его заново. 😊"
	} else {
		msg += "<strong>Текст сообщения:</strong> \n\n" + user.MailingMessagetext + "\n\nЕсли в тексте ошибка, то пришлите его заново. 😊"
	}

	return msg
}

func SendStatistic(b *Bot, update tgbotapi.Update, action CallBackAction) error {
	err := IsAdmin(b, update)
	if err != nil {
		return err
	}

	mailing, statuses, err := Admin.GetMailingWithStatuses(action.MailingID)
	if err != nil {
		b.SendMessage(tgbotapi.NewMessage(update.FromChat().ID, "Не удалось получить статистику: "+err.Error()))
		return err
	}
	msgText := "<strong>Рассылка</strong>\n"
	msgText += "<strong>ID: " + fmt.Sprint(mailing.ID) + "</strong>\n"
	if mailing.MailingType != MailingTeamChat.String() {
		msgText += "<strong>Когорта:</strong>  " + mailing.CohortName + "\n"
	}
	msgText += "<strong>Тип рассылки:</strong>  " + mailing.MailingType + "\n"
	if mailing.MessageText == "" {
		msgText += "\n<strong>Текст сообщения пуст</strong>\n " + "\n"
	} else {
		msgText += "<strong>Текст сообщения:</strong> \n\n" + mailing.MessageText + "\n\n"
	}
	msgText += "Сводка по пользователям:\n"

	var messages []string
	currentMessage := msgText

	for _, v := range statuses {
		line := v.UserName

		switch {
		case v.MsgIsSent && v.MsgIsReacted:
			line += " " + "✅ дошло с реакцией"
		case v.MsgIsSent && !v.MsgIsReacted:
			line += " " + "🟧 дошло без реакцией"
		case v.SendFailed:
			line += " " + "🟥 не дошло"
		default:
			line += " " + "❌еще не отправлено"
		}
		line += "\n"

		// Проверяем, помещается ли строка в текущее сообщение
		if len(currentMessage)+len(line) > 4096 {
			messages = append(messages, currentMessage)
			currentMessage = line // Начинаем новое сообщение с текущей строкой
		} else {
			currentMessage += line // Добавляем строку к текущему сообщению
		}
	}

	// Добавляем последнее сообщение, если оно не пустое
	if currentMessage != "" {
		messages = append(messages, currentMessage)
	}

	// Отправка сообщений (предполагается, что есть функция для отправки)
	for _, msgText := range messages {
		msg := tgbotapi.NewMessage(update.SentFrom().ID, msgText)
		msg.ParseMode = "html"
		b.SendMessage(msg)
	}
	return nil
}

func SendStatisticWithButtonDelete(b *Bot, update tgbotapi.Update, action CallBackAction) error {
	err := IsAdmin(b, update)
	if err != nil {
		return err
	}

	mailing, statuses, err := Admin.GetMailingWithStatuses(action.MailingID)
	if err != nil {
		return err
	}
	msgText := "<strong>Рассылка</strong>\n"
	msgText += "<strong>ID: " + fmt.Sprint(mailing.ID) + "</strong>\n"
	if mailing.MailingType != MailingTeamChat.String() {
		msgText += "<strong>Когорта:</strong>  " + mailing.CohortName + "\n"
	}
	msgText += "<strong>Тип рассылки:</strong>  " + mailing.MailingType + "\n"
	if mailing.MessageText == "" {
		msgText += "\n<strong>Текст сообщения пуст</strong>\n " + "\n"
	} else {
		msgText += "<strong>Текст сообщения:</strong> \n\n" + mailing.MessageText + "\n\n"
	}
	msgText += "Сводка по пользователям:\n"

	var messages []string
	currentMessage := msgText

	for _, v := range statuses {
		line := v.UserName

		if v.MsgIsSent && v.MsgIsReacted {
			line += " " + "✅ дошло с реакцией"
		}
		if v.MsgIsSent && !v.MsgIsReacted {
			line += " " + "🟧 дошло без реакцией"
		}
		if v.SendFailed {
			line += " " + "🟥 не дошло"
		}
		line += "\n"

		// Проверяем, помещается ли строка в текущее сообщение
		if len(currentMessage)+len(line) > 4096 {
			messages = append(messages, currentMessage)
			currentMessage = line // Начинаем новое сообщение с текущей строкой
		} else {
			currentMessage += line // Добавляем строку к текущему сообщению
		}
	}

	// Добавляем последнее сообщение, если оно не пустое
	if currentMessage != "" {
		messages = append(messages, currentMessage)
	}

	// Отправка сообщений (предполагается, что есть функция для отправки)
	for _, msgText := range messages {
		msg := tgbotapi.NewMessage(update.SentFrom().ID, msgText)
		msg.ParseMode = "html"
		msg.ReplyMarkup = CreateInlineKeyboard([][]ButtonData{{
			ButtonData{
				Text: "Отменить рассылку",
				Data: fmt.Sprintf(`{"ActionType":"StatisticDeleting","MailingID":%d}`, mailing.ID),
			}}})

		b.SendMessage(msg)
	}
	return nil
}

func SendStatisticDeleting(b *Bot, update tgbotapi.Update, action CallBackAction) error {
	err := IsAdmin(b, update)
	if err != nil {
		return err
	}

	mailing, statuses, err := Admin.GetMailingWithStatuses(action.MailingID)
	if err != nil {
		return err
	}

	for _, v := range statuses {
		if v.SendFailed {
			continue
		}

		deleteMsg := tgbotapi.NewDeleteMessage(v.TgID, v.MsgID)
		b.DeleteMessage(deleteMsg)
	}

	err = DeleteMailing(mailing.ID)
	if err != nil {
		logger.Error("Не удалось отменить рассылку: " + err.Error())
		b.SendMessage(tgbotapi.NewMessage(update.FromChat().ID, "Не удалось отменить рассылку: "+err.Error()))
		return nil
	}
	b.SendMessage(tgbotapi.NewMessage(update.FromChat().ID, "🗑️Рассылка с ID "+fmt.Sprint(mailing.ID)+" успешно удалена"))

	return nil
}

func SendReact(b *Bot, update tgbotapi.Update, action CallBackAction) error {
	status, err := Admin.UpdateStatusReaction(action.StatusID)
	if err != nil {
		return err
	}

	mailing, _, err := Admin.GetMailingWithStatuses(action.MailingID)
	if err != nil {
		return err
	}

	newMsgText := mailing.MessageText + "\n\n✅Вы отреагировали на сообщение. Спасибо!✅"

	editMsg := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, status.MsgID, newMsgText)
	_, err = b.EditMessage(editMsg)
	if err != nil {
		return err
	}

	return nil
}

var handleActiveMailings = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		mailings, err := Admin.GetAllMailings()
		if err != nil {
			return fmt.Errorf("failed to get mailings: %w", err)
		}

		msgText := "<strong>📨Активные рассылки</strong>"
		if len(mailings) == 0 {
			msgText += "\nАктивных рассылок нет"
		}

		msg := tgbotapi.NewMessage(update.FromChat().ID, msgText)
		msg.ParseMode = "html"

		// Создаем кнопки для каждой рассылки, по 3 в ряду
		if len(mailings) > 0 {
			var buttons [][]ButtonData
			var row []ButtonData

			for i, mailing := range mailings {
				// Формируем текст кнопки
				buttonText := fmt.Sprintf("ID:%d | %s", mailing.ID, mailing.CohortName)
				if len(buttonText) > 30 {
					buttonText = buttonText[:27] + "..."
				}

				// Создаем кнопку
				button := ButtonData{
					Text: buttonText,
					Data: fmt.Sprintf(`{"ActionType":"StatisticWithButtonDelete","MailingID":%d}`, mailing.ID),
				}

				row = append(row, button)

				// Если в ряду 3 кнопки или это последняя рассылка, добавляем ряд
				if len(row) == 3 || i == len(mailings)-1 {
					buttons = append(buttons, row)
					row = []ButtonData{}
				}
			}

			msg.ReplyMarkup = CreateInlineKeyboard(buttons)
		}

		// Отправляем сообщение
		_, err = b.SendMessage(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		return nil
	},
	Description: "Присылает список активных рассылок",
}

var HandleUpdateData = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		msg := tgbotapi.NewMessage(update.FromChat().ID, "Какие данные обновить?")
		msg.ParseMode = "html"
		msg.ReplyMarkup = CreateInlineKeyboard([][]ButtonData{{
			ButtonData{
				Text: "Данные веб форм",
				Data: `{"ActionType":"update","updateType":"webForm"}`,
			},
			ButtonData{
				Text: "Данные админов, кагорт.",
				Data: `{"ActionType":"update","updateType":"admins"}`,
			},
		}})

		_, err := b.sendMessage(msg)
		if err != nil {
			logger.Error("Ошибка при отправке сообщения: ", err)
			return err
		}

		return nil
	},
	Description: "Присыла��т список доступных данных для обновления и предлагает выбрать тип для обновления",
}

func updateData(b *Bot, update tgbotapi.Update, action CallBackAction) error {
	callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := b.botAPI.Request(callback); err != nil {
		logger.Error("Ошибка при отправке ответа на callback: ", err)
		return err
	}

	msgText := "Начался процесс обновления данных."
	if action.UpdateType == "admins" {
		msgText += "\n\nМеню админа будет недоступно некоторое время"
	} else {
		msgText += "\n\nФорма будет недоступна некоторое время"
	}

	msg := tgbotapi.NewMessage(update.FromChat().ID, msgText)
	sendedMsg, err := b.SendMessage(msg)
	if err != nil {
		logger.Error("Ошибка при отправке сообщения: ", err)
		return err
	}

	switch action.UpdateType {
	case "webForm":
		err = googlesheet.ColectSelectData(googlesheet.GoogleSheet)
	case "admins":
		err = googlesheet.ColectAdminsData(googlesheet.GoogleSheet)
	default:
		err = fmt.Errorf("неизвестный тип данных")
	}

	msg = tgbotapi.NewMessage(update.FromChat().ID, "Данные успешно обновлены.")
	if err != nil {
		msg.Text = "Ошибка при обновлении данных: " + err.Error()
	} else {
		msg.ReplyToMessageID = sendedMsg.MessageID
	}
	_, err = b.SendMessage(msg)
	if err != nil {
		logger.Error("Ошибка при отправке сообщения: ", err)
		return err
	}

	return nil
}

var HandleReloadBot = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		b.SendMessage(tgbotapi.NewMessage(update.SentFrom().ID, "Бот перезапускается."))

		os.Exit(1)

		return nil
	},
	Description: "Перезапускает бота. Т.к. бот обьеден с формой, то форма будет недоступна некоторое время",
}

var HandleNewMailing = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		b.SetUserState(update.SentFrom().ID, ChoosingMailingType)

		buttons := []string{"ЛС", "Чат с менеджером", "Командный чат", "Назад", "Закрыть"}
		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Выберите тип рассылки")
		msg.ReplyMarkup = CreateKeyboard(buttons, 2)
		b.SendMessage(msg)

		return nil
	},
	Description: "Присылает список доступных кагорт и предлагает выбрать одну для рассылки.",
}

var HandleBirthDays = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		b.SendMessage(tgbotapi.NewMessage(update.SentFrom().ID, "Начат процесс сборки данных."))

		birthDays, err := googlesheet.CollectBirthDays(googlesheet.GoogleSheet)
		if err != nil {
			msg := tgbotapi.NewMessage(update.SentFrom().ID, "Ошибка при получении информации: "+err.Error())
			b.SendMessage(msg)
			return err
		}
		if birthDays == nil || len(*birthDays) == 0 {
			msg := tgbotapi.NewMessage(update.SentFrom().ID, "Сегодня нет Дней Рождений")
			b.SendMessage(msg)
			return nil
		}

		msg := "🎉 <strong>Сегодня Дни Рождения:</strong>\n\n"

		for i := 0; i < len(*birthDays); i++ {
			birthDay := (*birthDays)[i]
			msg += fmt.Sprintf("👤 <strong>Имя:</strong> %s\n", birthDay.Name)
			msg += fmt.Sprintf("🆔 <strong>Пользователь:</strong> %s\n", birthDay.UserName)
			msg += fmt.Sprintf("🎂 <strong>Дата рождения:</strong> %s\n", birthDay.Date)
			msg += fmt.Sprintf("💼 <strong>Опыт работы:</strong> %s\n", birthDay.Experience) // Добавлено поле опыта
			msg += "-------------------------\n"

			if len(msg) > 4096 {
				msgg := tgbotapi.NewMessage(update.SentFrom().ID, msg)
				msgg.ParseMode = "html"

				b.SendMessage(msgg)
				msg = ""
			}
		}

		if len(msg) > 0 {
			msgg := tgbotapi.NewMessage(update.SentFrom().ID, msg)
			msgg.ParseMode = "html"

			b.SendMessage(msgg)
		}

		return nil
	},
	Description: "Присылает список Дней Рождений препопадавателей за текущую дату",
}

func HandleBirthDaysToTopic(b *Bot) error {
	conf := config.File.TelegramConfig

	birthDays, err := googlesheet.CollectBirthDays(googlesheet.GoogleSheet)
	if err != nil {
		msg := tgbotapi.NewMessage(conf.BotTgChat, "Ошибка при получении информации: "+err.Error())
		msg.ReplyToMessageID = conf.BirthTopicId
		b.SendMessage(msg)
		return err
	}
	if birthDays == nil || len(*birthDays) == 0 {
		msg := tgbotapi.NewMessage(conf.BotTgChat, "Сегодня нет Дней Рождений")
		msg.ReplyToMessageID = conf.BirthTopicId
		b.SendMessage(msg)
		return nil
	}

	msg := "🎉 <strong>Сегодня Дни Рождения:</strong>\n\n"

	for i := 0; i < len(*birthDays); i++ {
		birthDay := (*birthDays)[i]
		msg += fmt.Sprintf("👤 <strong>Имя:</strong> %s\n", birthDay.Name)
		msg += fmt.Sprintf("🆔 <strong>Пользователь:</strong> %s\n", birthDay.UserName)
		msg += fmt.Sprintf("🎂 <strong>Дата рождения:</strong> %s\n", birthDay.Date)
		msg += fmt.Sprintf("💼 <strong>Опыт работы:</strong> %s\n", birthDay.Experience) // Добавлено поле опыта
		msg += "-------------------------\n"

		if len(msg) > 4096 {
			msgg := tgbotapi.NewMessage(conf.BotTgChat, msg)
			msgg.ReplyToMessageID = conf.BirthTopicId
			msgg.ParseMode = "html"
			b.SendMessage(msgg)
			msg = ""
		}
	}
	msgg := tgbotapi.NewMessage(conf.BotTgChat, msg)
	msgg.ReplyToMessageID = conf.BirthTopicId
	msgg.ParseMode = "html"
	b.SendMessage(msgg)

	return nil
}

var HandleBackChoosingCohortsType = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		b.SetUserState(update.SentFrom().ID, ChoosingMailingType)

		buttons := []string{"ЛС", "Чат с менеджером", "Командный чат", "Закрыть"}
		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Выберите тип рассылки")
		msg.ReplyMarkup = CreateKeyboard(buttons, 2)
		b.SendMessage(msg)
		user := b.sessions.Get(update.SentFrom().ID)
		user.MailingMessagetext = ""
		user.CohortsName = ""
		user.MailingType = -1
		b.sessions.Set(update.SentFrom().ID, user)
		return nil
	},
	Description: "Возвращает меню на предыдущий шаг",
}

var HandleBackEnterMailingType = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {

		userID := update.SentFrom().ID
		user := b.sessions.Get(userID)

		if user.MailingType == MailingTeamChat {
			b.SetUserState(update.SentFrom().ID, ChoosingMailingType)

			buttons := []string{"ЛС", "Чат с менеджером", "Командный чат", "Назад", "Закрыть"}
			msg := tgbotapi.NewMessage(update.SentFrom().ID, "Выберите тип рассылки")
			msg.ReplyMarkup = CreateKeyboard(buttons, 2)
			b.SendMessage(msg)
			user.MailingMessagetext = ""
			user.CohortsName = ""
			user.MailingType = -1
			b.sessions.Set(userID, user)
			return nil
		}

		b.SetUserState(update.SentFrom().ID, ChoosingCohortsType)

		cohorts := append(cache.TelegramCacheApp.GetCohortsNames(), "Назад", "зак��ыть")
		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Выберите кагорту для рассылки")
		msg.ReplyMarkup = CreateKeyboard(cohorts, 4)

		b.SendMessage(msg)
		user.MailingMessagetext = ""
		user.CohortsName = ""
		b.sessions.Set(userID, user)
		return nil
	},
	Description: "Возвращает меню на предыдущий шаг",
}

var HandleSendMailing = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {

		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Рассылка началась.\nВас уведомят об окончании.\nЧерез 24 часа вам придет список проигнорировавших сообщение.")
		sendedMsg, _ := b.SendMessage(msg)

		userID := update.SentFrom().ID
		user := b.sessions.Get(userID)
		/////////////////////
		// TelegramBot.SendMessage(tgbotapi.NewMessage(userID, fmt.Sprint(b.sessions.GetAll())))
		////////////////////

		b.SetUserState(update.SentFrom().ID, MenuStateName)
		HandleMenu.Func(b, update)

		statuses, err := createMailingStatuses(user)
		if err != nil {
			b.SendMessage(tgbotapi.NewMessage(update.FromChat().ID, "Рассылка не удалась: "+err.Error()))
			return err
		}

		mailing := m.Mailing{
			AuthorTgID:      update.SentFrom().ID,
			MailingType:     user.MailingType.String(),
			CohortName:      user.CohortsName,
			MessageText:     user.MailingMessagetext,
			MailingStatuses: statuses,
			// Entities:        user.Entities,
		}

		if user.MailingType == MailingPrivateMessage {
			mailing.Button = true
		}
		id, err := Admin.CreateMailing(mailing)
		if err != nil {
			b.SendMessage(tgbotapi.NewMessage(update.FromChat().ID, "Рассылка не удалась: "+err.Error()))
			return err
		}

		msg = tgbotapi.NewMessage(update.FromChat().ID, `<strong>ID рассылки: `+strconv.FormatInt(id, 10)+`</strong>`)
		msg.ReplyToMessageID = sendedMsg.MessageID
		msg.ParseMode = tgbotapi.ModeHTML
		b.SendMessage(msg)

		return nil
	},
	Description: "Отправляет рассылку",
}

var HandleClose = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Меню закрыто")
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		b.SendMessage(msg)
		b.SetUserState(update.SentFrom().ID, "")
		return nil
	},
	Description: "Закрывает меню",
}

var HandleHelp = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		err := IsAdmin(b, update)
		if err != nil {
			return err
		}

		msgText := "Все команды:\n\n"

		msg := tgbotapi.NewMessage(update.SentFrom().ID, msgText)

		i := 1

		for _, state := range b.states {
			for k, v := range state.MessageRoute {

				msgText += fmt.Sprintf("<strong>%d</strong>)  %s - %s\n", i, k, v.Description)
				i++
			}
		}

		msg.Text = msgText
		msg.ParseMode = tgbotapi.ModeHTML
		b.SendMessage(msg)

		return nil
	},
	Description: "Присылает список всех команд",
}

var GlobalCatchAllCallBack = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {

		if update.CallbackQuery == nil {
			logger.Error("В CatchAllFunc update.CallbackQuery=nil")
			return fmt.Errorf("в CatchAllFunc update.CallbackQuery=nil")
		}

		var action CallBackAction
		err := json.Unmarshal([]byte(update.CallbackQuery.Data), &action)
		if err != nil {
			logger.Error("Ошибка при парсинге события callback: ", err)
			b.SendMessage(tgbotapi.NewMessage(update.SentFrom().ID, "Ошибка при парсинге события callback: "+err.Error()))
			return err
		}

		switch action.ActionType {
		case "Mailing":
			err = SendReact(b, update, action)
		case "Statistic":
			err = SendStatistic(b, update, action)

		case "Notification":
			if b.HandlCallbackLessonButton != nil {
				err = b.HandlCallbackLessonButton(b, update, action.NotificationUID)
			}
		case "StatisticDeleting":
			err = SendStatisticDeleting(b, update, action)
		case "StatisticWithButtonDelete":
			err = SendStatisticWithButtonDelete(b, update, action)
		case "update":
			err = updateData(b, update, action)
		default:
			b.ShowAlert(update.CallbackQuery.ID, "Неизвестная команда")
			// Отвечаем на callback query
			return fmt.Errorf("поступило неизвестное событие")
		}
		callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
		b.botAPI.Request(callback)

		if err != nil {
			logger.Error("Ошибка при обработке события callback: ", err)
			// b.SendMessage(tgbotapi.NewMessage(update.FromChat().ID, err.Error()))
			return err
		}

		return nil
	},
	Description: "CatchAllCallBack",
}

var ChoosingCohortsTypeCatchAllCallBack = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		if strings.ToLower(strings.TrimSpace(update.Message.Text)) == "закрыть" || strings.ToLower(strings.TrimSpace(update.Message.Text)) == "назад" {
			return nil
		}

		cohort := cache.TelegramCacheApp.GetCohortByName(update.Message.Text)

		if len(cohort) == 0 {
			b.SendMessage(tgbotapi.NewMessage(update.SentFrom().ID, "Кагорта с именем \""+update.Message.Text+"\" "+"не найдена"))
			return nil
		}

		userID := update.SentFrom().ID
		user := b.sessions.Get(userID)
		user.CohortsName = update.Message.Text
		b.sessions.Set(userID, user)

		b.SetUserState(update.SentFrom().ID, EnterMailingType)

		msg := tgbotapi.NewMessage(update.SentFrom().ID, "Введите  текст сообщения")
		msg.ReplyMarkup = CreateKeyboard([]string{"Закрыть", "Назад"}, 2)

		b.SendMessage(msg)

		return nil
	},
	Description: "Выбирает кагорту для рассылки",
}

func ConvertMessageToHTML(msg *tgbotapi.Message) string {
	if msg == nil || msg.Text == "" {
		return ""
	}

	// Преобразуем текст в слайс рун для безопасной работы с символами
	textRunes := []rune(msg.Text)
	result := ""
	lastOffset := 0

	for _, entity := range msg.Entities {
		// Получаем начальный и конечный индексы сущности
		start := int(entity.Offset)
		end := int(entity.Offset + entity.Length)

		// Добавляем текст до текущей сущности
		if start > lastOffset {
			result += string(textRunes[lastOffset:start])
		}

		// Получаем текст сущности
		entityText := string(textRunes[start:end])

		// Добавляем HTML-теги для сущности
		switch entity.Type {
		case "bold":
			result += "<strong>" + entityText + "</strong>"
		case "italic":
			result += "<i>" + entityText + "</i>"
		case "underline":
			result += "<u>" + entityText + "</u>"
		case "strikethrough":
			result += "<s>" + entityText + "</s>"
		case "code":
			result += "<code>" + entityText + "</code>"
		case "pre":
			result += "<pre>" + entityText + "</pre>"
		case "text_link":
			if entity.URL != "" {
				result += "<a href='" + entity.URL + "'>" + entityText + "</a>"
			} else {
				result += entityText
			}
		case "spoiler":
			// Telegram скрывает спойлеры с использованием класса tg-spoiler
			result += "<span class='tg-spoiler'>" + entityText + "</span>"
		case "blockquote":
			// Добавляем цитату как блок
			result += "<blockquote>" + entityText + "</blockquote>"
		default:
			// Просто добавляем текст, если тип неизвестен
			result += entityText
		}

		// Обновляем последний обработанный offset
		lastOffset = end
	}

	// Добавляем текст после последней сущности
	if lastOffset < len(textRunes) {
		result += string(textRunes[lastOffset:])
	}

	return result
}

var EnterMailingTypeCatchAllCallBack = Handler{
	Func: func(b *Bot, update tgbotapi.Update) error {
		msgText := update.Message.Text

		// Проверяем, является ли текст командой "закрыть" или "отправить"
		if msgText == "закрыть" || msgText == "отправить" {
			// Если это команда, возвращаем nil
			return nil
		}

		// Пропускаем любые другие слова
		userID := update.SentFrom().ID
		user := b.sessions.Get(userID)
		user.MailingMessagetext = ConvertMessageToHTML(update.Message)
		b.sessions.Set(userID, user)

		// Если сообщение не готово, то присылаем, то что есть и "Закрыть"
		if user.MailingMessagetext != "" && user.MailingType.String() != "" {
			// Если рассылка НЕ в командные чаты И когорты нет
			if user.MailingType != MailingTeamChat && user.CohortsName == "" {
				return nil
			}
			msg := tgbotapi.NewMessage(update.SentFrom().ID, formatReadyMailMsg(user))
			msg.ReplyMarkup = CreateKeyboard([]string{"Отправить", "Назад", "Закрыть"}, 2)
			msg.ParseMode = "html"
			b.SendMessage(msg)
		} else {
			// Если сообщение готово, то присылаем "Отправить", "Закрыть"
			msg := tgbotapi.NewMessage(update.SentFrom().ID, formatReadyMailMsg(user))
			msg.ReplyMarkup = CreateKeyboard([]string{"Закрыть", "Назад"}, 2)
			msg.ParseMode = "html"
			b.SendMessage(msg)
		}

		return nil
	},
	Description: "Вводит текст сообщения для рассылки",
}

// По хорошему исползовать действие при входе.
// Реализовать функцию, которая перенаправляет "пользователя" в другое состояние с использованием контекста и функции при входе

var UserStates map[string]State = map[string]State{
	GlobalState: {
		Global:            true,
		NoContext:         true,
		NotEntranceAction: true,
		CatchAll:          true,
		CatchAllCallBack:  true,
		CatchAllFunc: Handler{
			Func: func(b *Bot, update tgbotapi.Update) error {
				HandleStatDate.Func(b, update)
				return nil
			},
		},
		CatchAllCallBackfunc: GlobalCatchAllCallBack,
		AtEntranceFunc:       defaultHandler,

		MessageRoute: map[string]Handler{
			"/start":     HandleStartMessage,
			"/tdstat":    HandleGetTodayStatistics,
			"/ydstat":    HandleGetYesterdayStatistics,
			"/forms":     HandleStat,
			"/menu":      HandleMenu,
			"меню":       HandleMenu,
			"закрыть":    HandleClose,
			"/mylessons": HandleGetMyLessons,
			"мои уроки":  HandleGetMyLessons,
			"/dr":        HandleBirthDays,
			"/help":      HandleHelp,
		},
		CallBackRoute: map[string]HandlerFunc{},
	},
	MenuStateName: {
		Global:            false,
		NoContext:         false,
		NotEntranceAction: true,
		CatchAll:          false,
		CatchAllCallBack:  false,

		CatchAllFunc:         defaultHandler,
		CatchAllCallBackfunc: defaultHandler,
		AtEntranceFunc:       defaultHandler,

		MessageRoute: map[string]Handler{
			"перезагрузить бота":  HandleReloadBot,
			"новая рассылка":      HandleNewMailing,
			"активные рассылки":   handleActiveMailings,
			"обновить данные":     HandleUpdateData,
			"др препопадавателей": HandleBirthDays,
		},
		CallBackRoute: map[string]HandlerFunc{},
	},
	ChoosingMailingType: {
		Global:            false,
		NoContext:         false,
		NotEntranceAction: true,
		CatchAll:          false,
		CatchAllCallBack:  false,

		CatchAllFunc:         defaultHandler,
		CatchAllCallBackfunc: defaultHandler,
		AtEntranceFunc:       defaultHandler,

		MessageRoute: map[string]Handler{
			"лс":               ActionOnMailingType(MailingPrivateMessage),
			"чат с менеджером": ActionOnMailingType(MailingManagerChat),
			"командный чат":    ActionOnMailingType(MailingTeamChat),
			"назад":            HandleMenu,
		},
		CallBackRoute: map[string]HandlerFunc{},
	},
	ChoosingCohortsType: {
		Global:            false,
		NoContext:         false,
		NotEntranceAction: true,
		CatchAll:          true,
		CatchAllCallBack:  false,

		CatchAllCallBackfunc: defaultHandler,
		AtEntranceFunc:       defaultHandler,
		CatchAllFunc:         ChoosingCohortsTypeCatchAllCallBack,

		MessageRoute: map[string]Handler{
			"назад": HandleBackChoosingCohortsType,
		},
		CallBackRoute: map[string]HandlerFunc{},
	},
	EnterMailingType: {
		Global:            false,
		NoContext:         false,
		NotEntranceAction: true,
		CatchAll:          true,
		CatchAllCallBack:  false,

		CatchAllCallBackfunc: defaultHandler,

		AtEntranceFunc: defaultHandler,
		CatchAllFunc:   EnterMailingTypeCatchAllCallBack,

		MessageRoute: map[string]Handler{
			"отправить": HandleSendMailing,
			"назад":     HandleBackEnterMailingType,
		},
		CallBackRoute: map[string]HandlerFunc{},
	},
}
