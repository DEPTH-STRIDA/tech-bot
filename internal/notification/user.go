package notification

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/easycodeapi"
	"easycodeapp/internal/infrastructure/logger"
	m "easycodeapp/internal/model"
	"easycodeapp/internal/tg"
	easycodeapipkg "easycodeapp/pkg/easycodeapi"
	"easycodeapp/pkg/model"
	"errors"
	"fmt"
	"math/rand"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartHandleMorningMessage (утреннее сообщение об уроках за весь день) получает всех преподавателей из БД, запрашивает уроки по CRM id из бд, отправка
func (app *NotificationManager) UpdateLessonDB(isNightRun bool) {
	// Получение всех crm_id
	var crmIds []int64
	err := app.User.GetColumnValues("crm_id", &crmIds)
	if err != nil {
		logger.Error("Ошибка при получении всех crm_id: ", err)
		return
	}

	logger.Debug("Получены все crmd_ids: ", crmIds)
	app.updateLessonDB(crmIds, isNightRun)
}

// updateLessonDB обновляет инфорцию об уроках в БД (message).
// Выполняется 2 запроса к API для всего списка преподавателей.
func (app *NotificationManager) updateLessonDB(teachers []int64, isNightRun bool) {
	logger.Info("ОБНОВЛЕНИЕ ДАННЫХ СПИСКА УРОКОВ (СООБЩЕНИЯ)")
	length := len(teachers)
	mid := length / 2
	currentDate := GetCurrentDateInFormat()
	if len(teachers[:mid]) != 0 {
		easycodeapi.Api.GettAllLessons(teachers[:mid], currentDate, isNightRun, app.AddLessonsToDB)
	}
	if len(teachers[mid:]) != 0 {
		easycodeapi.Api.GettAllLessons(teachers[mid:], currentDate, isNightRun, app.AddLessonsToDB)
	}
}

// AddLessonsToDB отправляет уроки в БД. Используется как handler.
// Преобразует параметр lessons в структуру, которая может записаться в БД (message).
func (app *NotificationManager) AddLessonsToDB(lessons []easycodeapipkg.Lesson, isNightRun bool) error {
	if len(lessons) == 0 {
		logger.Error("Не удалось добавить уроки в БД: len(lessons)==0")
		return errors.New("длина lessons 0")
	}

	for _, lesson := range lessons {
		var user model.User
		err := app.User.GetRowByColumn("crm_id", lesson.TeacherID, &user)
		if err != nil {
			logger.Error("Не удалось получить преподавателя из базы данных по CRM_ID: ", lesson.TeacherID)
			continue
		}

		// Если это дневной прогон, выполняем проверку на изменения
		if !isNightRun {
			app.processLessonChanges(user, lesson)
		}

		// Получаем дату/время, когда надо отправить сообщение
		sendDateTime, err := calculateTime(lesson.DateStart, time.Duration(config.File.NotificationConfig.BeforeLessonNotificationTime)*time.Minute)
		if err != nil {
			logger.Error("Не удалось получить время за 30 минут до начала урока: ", err)
			continue
		}

		// Получение текущей даты и времени
		currentTime := time.Now()

		// Сравнение дат по году, месяцу и дню
		if !isSameYearMonthDay(sendDateTime, currentTime) {
			logger.Info("Попытка вставить в message table БД урок не за текущую дату sendDateTime = ", sendDateTime, ";currentTime= ", currentTime)
			continue
		}

		// Получаем дату/время в корректном формате
		lessonDateTime, err := calculateTime(lesson.DateStart, 0*time.Second)
		if err != nil {
			logger.Error("Не удалось получить дату/время урока в удобном формате: ", err)
			continue
		}

		layout := "2006-01-02 15:04:05"
		parsedTime, err := time.Parse(layout, lesson.DateStart)
		if err != nil {
			logger.Error("Ошибка парсинга времени: ", err)
			return err
		}
		newTime := parsedTime.Add(time.Duration(config.File.NotificationConfig.DelayTime) * time.Minute)

		message := m.Message{
			UserName:                 user.UserName,
			ChatID:                   user.ChatID,
			LessonID:                 lesson.LessonID,
			MsgSendTime:              sendDateTime,
			MsgIssent:                false,
			MsgIsPressed:             false,
			MsgIsMorningNotification: false,
			CourseName:               lesson.Name,
			CourseID:                 int(lesson.CourseID),
			CRMID:                    lesson.TeacherID,
			TeacherName:              user.TeacherName,
			ActiveMember:             lesson.ActiveMemberCount,
			LessonNumber:             int(lesson.LessonNumber),
			LessonTime:               lessonDateTime,
			UID:                      GenerateID(4),
			DelayTime:                newTime,
		}

		err = app.Message.InsertRowUnique(&message, map[string]string{
			"CourseName": "course_name",
			"LessonTime": "lesson_time",
		})
		if err != nil {
			logger.Error("Ошибка при отправке уроков преподавателя в БД: ", err, " ---- message", message)
		}

		cachedMessage := m.CachedMessage{Message: message}
		err = app.Message.InsertRowUnique(&cachedMessage, map[string]string{
			"CourseName": "course_name",
			"LessonTime": "lesson_time",
		})
		if err != nil {
			logger.Error("Ошибка при отправке уроков преподавателя в БД: ", err, " ---- message", message)
		}
	}
	return nil
}

func (app *NotificationManager) processLessonChanges(user model.User, lesson easycodeapipkg.Lesson) {
	// Получаем текущие уроки из CachedMessage
	var cachedLessons []m.CachedMessage
	err := db.DB.Where("crm_id = ?", lesson.TeacherID).Find(&cachedLessons).Error
	if err != nil {
		logger.Error("Ошибка получения уроков из CachedMessage: ", err)
		return
	}

	// Создаем мапу для быстрого поиска
	cachedLessonMap := make(map[int64]m.CachedMessage)
	for _, cl := range cachedLessons {
		cachedLessonMap[cl.LessonID] = cl
	}

	// Проверяем, есть ли новый урок
	if _, exists := cachedLessonMap[lesson.LessonID]; !exists {
		// Отправляем уведомление о новом уроке
		app.sendNewLessonNotification(user, lesson)
	}

	// Удаляем из мапы, чтобы в конце остались только удаленные уроки
	delete(cachedLessonMap, lesson.LessonID)

	// Оставшиеся в мапе уроки считаются удаленными
	for _, cl := range cachedLessonMap {
		// Отправляем уведомление об удалении урока
		app.sendDeletedLessonNotification(user, cl)

		// Удаляем урок из базы данных
		err := db.DB.Delete(&cl).Error
		if err != nil {
			logger.Error("Ошибка при удалении урока из БД: ", err)
		}
	}
}

func (app *NotificationManager) sendNewLessonNotification(user model.User, lesson easycodeapipkg.Lesson) {
	// Логика отправки уведомления о новом уроке
	msgText := fmt.Sprintf("Новый урок: %s, %s", lesson.Name, lesson.DateStart)
	msg := tgbotapi.NewMessage(user.ChatID, msgText)
	_, err := tg.TelegramBot.SendMessage(msg)
	if err != nil {
		logger.Error("Ошибка при отправке уведомления о новом уроке: ", err)
	}
}

func (app *NotificationManager) sendDeletedLessonNotification(user model.User, lesson m.CachedMessage) {
	// Логика отправки уведомления об удалении урока
	// Логика отправки уведомления о новом уроке
	msgText := fmt.Sprintf("Урок: %s, %s удален", lesson.CourseName, lesson.LessonTime)
	msg := tgbotapi.NewMessage(user.ChatID, msgText)
	_, err := tg.TelegramBot.SendMessage(msg)
	if err != nil {
		logger.Error("Ошибка при отправке уведомления об удалении урока: ", err)
	}
}

// GenerateID возвращает сгенерированную строку указанной длины, состоящей из латинского алфавита и цифр
func GenerateID(idLength int) string {
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var idCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, idLength)
	for i := range b {
		b[i] = idCharset[seededRand.Intn(len(idCharset))]
	}
	return string(b)
}

// Функция для проверки, наступила ли уже дата без учета часового пояса
func hasDatePassed(targetTime time.Time) bool {
	// Приводим время к формату без учета часового пояса
	currentTime := time.Now().Local()
	targetTime = targetTime.UTC()
	// Сравниваем часы, минуты, секунды и наносекунды
	currentHour, currentMin, currentSec := currentTime.Clock()
	targetHour, targetMin, targetSec := targetTime.Clock()

	if currentHour > targetHour {
		return true
	} else if currentHour == targetHour && currentMin > targetMin {
		return true
	} else if currentHour == targetHour && currentMin == targetMin && currentSec >= targetSec {
		return true
	}
	return false
}

// Функция для проверки, совпадают ли год, месяц и день
func isSameYearMonthDay(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}

// GetCurrentDateInFormat возвращает текущую дату в формате "YYYY-MM-DDT00:00:00.000Z"
func GetCurrentDateInFormat() string {
	// Получаем текущее время в UTC
	now := time.Now()

	// Форматируем дату в нужный формат
	return now.Format("2006-01-02T00:00:00.000Z")
}

// calculateTime принимает строку даты-времени в формате "2006-01-02 15:04:05" и время для вычитания
func calculateTime(dateTimeStr string, minusTime time.Duration) (time.Time, error) {
	// Парсинг строки даты-времени в тип time.Time
	layout := "2006-01-02 15:04:05"
	parsedTime, err := time.Parse(layout, dateTimeStr)
	if err != nil {
		return time.Time{}, err
	}

	// Вычитание времени
	newTime := parsedTime.Add(-minusTime)

	return newTime, nil
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
