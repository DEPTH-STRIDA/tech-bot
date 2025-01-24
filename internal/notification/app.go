package notification

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/googlesheet"
	"easycodeapp/internal/infrastructure/logger"
	m "easycodeapp/internal/model"
	"easycodeapp/internal/tg"
	"easycodeapp/pkg/model"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

var NotificationApp *NotificationManager = &NotificationManager{}

func InitNotificationApp() error {
	var err error
	NotificationApp, err = NewNotificationManager()
	if err != nil {
		return err
	}

	return nil
}

// NotificationManager контролирует рабочий процесс, связанный с уведомлениями: отправку в тг, реакцию на кнопки
type NotificationManager struct {
	User     *Database
	Message  *Database
	CallBack *Database
}

// NewNotificationManager создает и возвращает экземпляр NotificationManager, запускает рабочие процессы связанные с обновлением
func NewNotificationManager() (*NotificationManager, error) {
	// Инициализация баз данных
	user, err := NewDatabase()
	if err != nil {
		return nil, err
	}

	message, err := NewDatabase()
	if err != nil {
		return nil, err
	}

	сallBack, err := NewDatabase()
	if err != nil {
		return nil, err
	}

	// Затем создаем новые таблицы
	err = сallBack.CreateTable(&CallBack{})
	if err != nil {
		logger.Info("Не удалось создать таблицу CallBack!")
		return nil, err
	}

	err = user.CreateTable(&model.User{})
	if err != nil {
		return nil, err
	}

	err = message.CreateTable(&m.Message{})
	if err != nil {
		return nil, err
	}

	err = message.CreateTable(&OldMessage{})
	if err != nil {
		return nil, err
	}

	notificationManager := &NotificationManager{
		User:     user,
		Message:  message,
		CallBack: сallBack,
	}

	// Установка универсального обработчика в телеграмм
	tg.TelegramBot.HandlCallbackLessonButton = notificationManager.HandlCallbackRoute

	if !config.File.WebConfig.IsTestMode {
		// Обновление БД преподавателей
		notificationManager.updateTeachersDB()

		// Удаление старых сообщений
		notificationManager.Message.HandleMessageTable("crm_id", 200, notificationManager.deleteOldMessage)

		//  Обновление уроков/сообщений в БД
		notificationManager.UpdateLessonDB(true)

		//  Переодичное обновление данных уроков
		go notificationManager.StartPeriodicLessonDBUpdate()

		// // Переодичное удаление старых сообщений
		go notificationManager.startPeriodicDeletionOldMessages(1 * time.Hour)

		// //  Переодичная отправка утренних сообщений
		go notificationManager.HandleMorningMessage()

		// //  Проверка на необходимость отправки сообщений из БД, проверка callback
		go notificationManager.StartDailyUpdate()

		// // Ежедневное обновление данных преподавателей
		go notificationManager.StartDailyUserTableRefresh()
	} else {
		logger.Info("ВКЛЮЧЕН ТЕСТОВЫЙ ЗАПУСК. Переодическое обновление данных уведомлений отключено")
	}

	return notificationManager, nil
}

// checkArrayLengths Функция для проверки длин массивов с подробным выводом ошибок
func checkArrayLengths(rangeInt int, crms, names, usernames, chat_ids []string) error {
	lengths := map[string]int{
		"rangeInt":  rangeInt,
		"crms":      len(crms),
		"names":     len(names),
		"usernames": len(usernames),
		"chat_ids":  len(chat_ids),
	}

	mismatch := false
	var errorDetails strings.Builder
	errorDetails.WriteString("Несовпадение длин массивов:\n")

	for name, length := range lengths {
		errorDetails.WriteString(fmt.Sprintf("- %s: %d\n", name, length))
		if length != rangeInt {
			mismatch = true
		}
	}

	if mismatch {
		return fmt.Errorf(errorDetails.String())
	}

	return nil
}

// GetUserData возвращает данные пользователей
func (app *NotificationManager) GetUsersData() ([]model.User, error) {
	conf := config.File.GoogleSheetConfig
	rangeStr, err := googlesheet.GoogleSheet.GetCellValue(conf.SelectDataTableID, conf.UsersListName, "C1")
	if err != nil {
		return nil, err
	}
	rangeInt, err := strconv.Atoi(rangeStr)
	if err != nil {
		return nil, err
	}

	matrix, err := googlesheet.GoogleSheet.GetMatrix(conf.SelectDataTableID, conf.UsersListName, 1, 4, 3, rangeInt)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных преподавателей: %w", err)
	}
	if len(matrix) != 4 {
		return nil, fmt.Errorf("ошибка при получении данных преподавателей: len(matrix) != 4. Len = %d", len(matrix))
	}
	for i := 0; i < len(matrix); i++ {
		if len(matrix[i]) != rangeInt-2 {
			return nil, fmt.Errorf("ошибка при получении данных преподавателей. Масиив %d не равен длине %d. Длина = %d", i, rangeInt-2, len(matrix[i]))
		}
	}

	crms := matrix[0]
	names := matrix[1]
	usernames := matrix[2]
	chatIds := matrix[3]

	// Использование в основном коде
	rangeInt -= 2 // Т.к. основные массивы идут A2:A, то их размер меньше на 1
	if err := checkArrayLengths(rangeInt, crms, names, usernames, chatIds); err != nil {
		logger.Error(err)
		return nil, err
	}

	users := []model.User{}

	errorsToTg := ""

	for i := 0; i < rangeInt; i++ {
		isValid := true

		crmId, err := strconv.ParseInt(crms[i], 10, 64)
		if err != nil {
			logger.Error("Не удалось получить crmId преподавателя", names[i], "\nОшибка: ", err.Error())
			isValid = false
		}
		if strings.TrimSpace(names[i]) == "" {
			logger.Error("Имя преподавателя не указано. CRM ID: ", crms[i])
			isValid = false
		}
		if strings.TrimSpace(usernames[i]) == "" {
			logger.Error("Username преподавателя не указано. CRM ID: ", crms[i])
			isValid = false
		}
		chatId, err := strconv.ParseInt(chatIds[i], 10, 64)
		if err != nil {
			logger.Error("chatId преподавателя не указано. CRM ID: ", crms[i])
			isValid = false
		}

		user := model.User{
			CRMID:       crmId,
			TeacherName: strings.TrimSpace(names[i]),
			UserName:    strings.TrimSpace(usernames[i]),
			ChatID:      chatId,
			// UserName: "@Tichomirov2003",
			// ChatID: 2024983086,
		}

		// В цикле обработки
		if isValid {
			users = append(users, user)
		} else {
			errorsToTg += names[i] + ", "
		}
	}

	if errorsToTg != "" {
		message := "<strong>🚨Ошибки при получении данных преподавателей</strong>\n" + errorsToTg
		tg.TelegramBot.SendAllAdmins(message)
	}

	if len(users) == 0 {
		tg.TelegramBot.SendAllAdmins("ни удалось получить ни одного пользователя после парсинга гугл таблиц")
		return nil, fmt.Errorf("ни удалось получить ни одного пользователя после парсинга гугл таблиц")
	}
	return users, nil
}

// updateTeachersDB обращается к таблицам и заполняет полученной информацией БД
func (app *NotificationManager) updateTeachersDB() error {
	users, err := app.GetUsersData()
	if err != nil {
		logger.Info("Ошибка при работе с таблицей преподавателей: ", err)
		return err
	}

	logger.Info("Получены данные преподавателей: ", users)

	for i := 0; i < len(users); i++ {
		err = app.UpsertUser(&users[i])
		if err != nil {
			logger.Error("updateTeachersDB: Ошибка при обновлении/создании преподавателя в БД: ", err)
			continue // Продолжаем с следующим пользователем даже при ошибке
		}
	}
	return nil
}

// UpsertUser обновляет существующего пользователя или создает нового
func (app *NotificationManager) UpsertUser(user *model.User) error {
	if user == nil {
		return errors.New("user is nil")
	}

	// Проверяем существование пользователя по username
	var existingUser model.User
	result := app.User.DB.Where("user_name = ?", user.UserName).First(&existingUser)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Пользователь не найден - создаем нового
			if err := app.User.DB.Create(user).Error; err != nil {
				return errors.New("failed to create new user: " + err.Error())
			}
			return nil
		}
		return errors.New("database error: " + result.Error.Error())
	}

	// Пользователь найден - обновляем все поля кроме username
	updates := map[string]interface{}{
		"crm_id":       user.CRMID,
		"teacher_name": user.TeacherName,
		"chat_id":      user.ChatID,
	}

	if err := app.User.DB.Model(&existingUser).Updates(updates).Error; err != nil {
		return errors.New("failed to update user: " + err.Error())
	}

	return nil
}

func addNegative100Prefix(num int64) int64 {
	str := fmt.Sprintf("-100%d", num)
	result, _ := strconv.ParseInt(str, 10, 64)
	return result
}

// startNadleOldMessages раз в 30 минут обрабатывает все сообщения. Старые переносятся в дргую БД. Очень старые, удаляются из старой БД.
func (app *NotificationManager) startPeriodicDeletionOldMessages(pause time.Duration) {
	for {
		app.Message.HandleMessageTable("crm_id", 200, app.deleteOldMessage)
		time.Sleep(pause)
	}
}

// handleOldMessage удаление отработанных заявок, перенос в старую таблицу.
func (app *NotificationManager) deleteOldMessage(results []m.Message) {
	for _, message := range results {
		if !isSameYearMonthDay(time.Now(), message.LessonTime) {

			err := app.Message.DeleteRecordByColumn("id", message.ID, m.Message{})
			if err != nil {
				logger.Info("Не  удалось удалить сообщение при проверке старых сообщений: ", err)
				continue
			}
			err = app.Message.DeleteRecordByColumn("id", message.ID, m.CachedMessage{})
			if err != nil {
				logger.Info("Не  удалось удалить сообщение при проверке старых сообщений: ", err)
				continue
			}
		}
	}
}

// SendLessonsNotification отправляет данные об уроках за день в чаты преподавателей
func (app *NotificationManager) SendLessonsNotification(messages []m.Message) {
	if len(messages) == 0 || messages == nil {
		return
	}

	text := fmt.Sprintf("<strong>🙂Имя преподавателя: %s</strong>\n\n", messages[0].TeacherName)
	for _, msg := range messages {
		text += fmt.Sprintf(`
		<strong>📋Название группы</strong>: %s
		<strong>⏰Время урока:</strong> %s

		`,
			msg.CourseName, msg.LessonTime.Format("2006-01-02 15:04"))
	}

	// Отправка с префиксом "-100"
	msg := tgbotapi.NewMessage(addNegative100Prefix(messages[0].ChatID), text)
	msg.ParseMode = "html"
	_, err := tg.TelegramBot.SendMessageRepetLowPriority(msg, 3)
	if err != nil {

		// Отправка без модификаций
		msg = tgbotapi.NewMessage(messages[0].ChatID, text)
		msg.ParseMode = "html"
		_, err = tg.TelegramBot.SendMessageRepetLowPriority(msg, 3)
		if err != nil {

			// Отправка с префиксом "-"
			msg = tgbotapi.NewMessage(messages[0].ChatID*-1, text)
			msg.ParseMode = "html"
			_, err = tg.TelegramBot.SendMessageRepetLowPriority(msg, 3)
			if err != nil {
				logger.Error("Ошибка при отправке сообщения с информацией о всех утренних сообщениях преподавателя: ", err, "\n; преподавателя: ", messages[0].TeacherName, messages[0].UserName)
				return
			}
		}
	}
}

// StartDailyUpdate запускает цикл, который бесконечно начнет отправлять, а затем проверять сообщения из message table БД
func (app *NotificationManager) StartDailyUpdate() {
	// необходима пауза, чтобы все таблицы в БД успели создаться/обновиться
	time.Sleep(3 * time.Second)
	for {
		logger.Info("Новая итерация проверка БД")
		app.Message.HandleMessageTable("crm_id", 200, app.handleCheckMessages)
		time.Sleep(time.Duration(config.File.NotificationConfig.AfterCheckpause) * time.Second)

		app.Message.HandleMessageTable("crm_id", 200, app.handleSendMessages)
		time.Sleep(time.Duration(config.File.NotificationConfig.AfterSendPause) * time.Second)
	}
}

func (app *NotificationManager) RefreshUserTable() error {

	// Очистка таблицы
	err := app.User.DropTable(&model.User{})
	if err != nil {
		return fmt.Errorf("ошибка при удалении таблицы пользователей: %w", err)
	}

	// Создание новой таблицы
	err = app.User.CreateTable(&model.User{})
	if err != nil {
		return fmt.Errorf("ошибка при создании новой таблицы пользователей: %w", err)
	}

	// Загрузка данных
	app.updateTeachersDB()

	return nil
}

func (app *NotificationManager) StartDailyUserTableRefresh() {
	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 23, 20, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}
		duration := nextRun.Sub(now)

		logger.Info("Следующее обновление таблицы пользователей запланировано на ", nextRun.Format("2006-01-02 15:04:05"))
		time.Sleep(duration)

		logger.Info("Начало обновления таблицы пользователей")
		err := app.RefreshUserTable()
		if err != nil {
			logger.Error("Ошибка при обновлении таблицы пользователей: ", err)
		} else {
			logger.Info("Таблица пользователей успешно обновлена")
		}
	}
}

// StartHandleMorningUser утром отправляет сообщение об уроках за день
func (app *NotificationManager) HandleMorningMessage() {
	for {
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())

		// Если текущее время после 6 утра, планируем на следующий день
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		// Ждем до следующего запланированного времени
		duration := nextRun.Sub(now)
		logger.Info("HandleMorningMessage: Время обновления будет приостановлено на ", duration, " до следующего запуска ", nextRun)
		time.Sleep(duration)

		// После ожидания запускаем обработку
		app.StartHandleMorningMessage()
	}
}

// StartHandleMorningMessage (утреннее сообщение об уроках за весь день) получает всех преподавателей из БД, запрашивает уроки по CRM id из бд, отправка
func (app *NotificationManager) StartHandleMorningMessage() {
	// Получение всех crm_id
	var crmIds []int
	err := app.User.GetColumnValues("crm_id", &crmIds)
	if err != nil {
		logger.Error("Ошибка при получении всех crm_id: ", err)
		return
	}

	logger.Debug("Получены все crmd_ids: ", crmIds)

	// Получение уроков по каждому crm_id
	for _, v := range crmIds {
		var messages []m.Message
		err := app.Message.GetRecordsByColumn("crm_id", v, &messages)
		if err != nil {
			logger.Error("Не удалось получить все значения по фильтру: ", err)
		}

		if len(messages) == 0 || messages == nil {
			continue
		}

		app.SendLessonsNotification(messages)
	}
}

// StartPeriodicLessonDBUpdate запускает периодическое обновление уроков в БД
func (app *NotificationManager) StartPeriodicLessonDBUpdate() {
	for {
		now := time.Now()

		// Определяем, является ли текущее обновление ночным (в полночь)
		isNightRun := now.Hour() == 0

		// Проверяем, если текущее время совпадает с одним из интервалов
		if now.Hour()%3 == 0 {
			logger.Info("Начало обновления уроков в БД")
			app.UpdateLessonDB(isNightRun)
		}

		// Рассчитываем время до следующего интервала
		nextRun := now.Add(time.Duration(3-now.Hour()%3) * time.Hour)
		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), nextRun.Hour(), 0, 0, 0, nextRun.Location())
		duration := nextRun.Sub(now)

		logger.Info("Следующее обновление уроков в БД запланировано на ", nextRun.Format("2006-01-02 15:04:05"))
		time.Sleep(duration)
	}
}
