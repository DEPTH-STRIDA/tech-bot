package web

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/tg"
	"easycodeapp/pkg/model"
	"easycodeapp/pkg/request"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"easycodeapp/internal/infrastructure/db"

	"github.com/gorilla/mux"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

var App *WebApp

func InitWebApp() error {
	var err error

	App, err = NewWebApp()
	if err != nil {
		return err
	}

	return nil
}

// WebApp веб приложение. Мозг программы, который использует большинство других приложений
type WebApp struct {
	Router        *mux.Router                   // Маршрутизатор
	TemplateCache map[string]*template.Template // Карта шаблонов

	reqSheet    *request.RequestHandler
	reqTelegram *request.RequestHandler
}

// NewWebApp создает и возвращает веб приложение
func NewWebApp() (*WebApp, error) {
	// Загрузка шаблонов
	templateCache, err := NewTemplateCache("../../ui/tech_bot/html/")
	if err != nil {
		return nil, err
	}

	reqSheet, err := request.NewRequestHandler(request.Config{
		BufferSize: 100,
		Logger:     logger.Log,
	})
	if err != nil {
		return nil, err
	}
	go reqSheet.ProcessRequests(1 * time.Second)

	reqTelegram, err := request.NewRequestHandler(request.Config{
		BufferSize: 100,
		Logger:     logger.Log,
	})
	if err != nil {
		return nil, err
	}
	go reqTelegram.ProcessRequests(1 * time.Second)

	app := WebApp{
		TemplateCache: templateCache,
		reqSheet:      reqSheet,
		reqTelegram:   reqTelegram,
	}
	// Установка параметров
	app.Router = app.SetRoutes()
	go app.StartPeriodSendingToGoogleSheet()
	return &app, nil
}

// HandleUpdates запускает HTTP сервер
func (app *WebApp) HandleUpdates() error {
	conf := config.File.WebConfig

	msg := "Бот запущен и готов к работе (" + conf.APPIP + ":" + conf.APPPORT + ")!\n(ง'̀-'́)ง"

	if conf.IsTestMode {
		msg += "\nВКЛЮЧЕН ТЕСТОВЫЙ ЗАПУСК. Переодическое обновление данных отключено"
	}

	logger.Info(msg)

	tg.TelegramBot.SendAllAdmins(msg)
	err := http.ListenAndServe(conf.APPIP+":"+conf.APPPORT, app.Router)
	if err != nil {
		return fmt.Errorf("ошибка при запуске сервера: %v", err)
	}
	return nil
}

// ConvertDateFormat конвертирует дату из различных форматов в "дд.мм.гггг"
func ConvertDateFormat(inputDate string) (string, error) {
	logger.Info("Конвертация: ", inputDate)
	// Список поддерживаемых форматов ввода
	formats := []string{
		"2006-01-02",                    // гггг-мм-дд
		"02.01.2006",                    // дд.мм.гггг
		"2006.01.02",                    // гггг.мм.дд
		"2006/01/02",                    // гггг/мм/дд
		"02/01/2006",                    // дд/мм/гггг
		"2006-01-02 15:04:05",           // с временем
		"2006-01-02T15:04:05Z",          // ISO формат
		"2006-01-02 15:04:05.999999-07", // с миллисекундами и таймзоной
		"2006-01-02 15:04:05.999999+07", // альтернативный формат таймзоны
	}

	var parsedDate time.Time
	var err error

	// Если входная строка пустая
	if inputDate == "" {
		return "", fmt.Errorf("пустая строка даты")
	}

	// Очистка строки от лишних пробелов
	inputDate = strings.TrimSpace(inputDate)

	// Пробуем все поддерживаемые форматы
	for _, format := range formats {
		parsedDate, err = time.Parse(format, inputDate)
		if err == nil {
			break
		}
	}

	// Если ни один формат не подошел
	if err != nil {
		return "", fmt.Errorf("неподдерживаемый формат даты: %s. Ошибка: %v", inputDate, err)
	}

	// Проверка валидности даты
	year := parsedDate.Year()
	if year < 1900 || year > 9999 {
		return "", fmt.Errorf("год %d вне допустимого диапазона (1900-9999)", year)
	}

	// Преобразование в требуемый формат
	formattedDate := parsedDate.Format("02.01.2006")

	return formattedDate, nil
}

// StartPeriodSendingToGoogleSheet раз в минуту отправляет данные из таблицы в Google Sheets
func (app *WebApp) StartPeriodSendingToGoogleSheet() {
	time.Sleep(2 * time.Minute)

	for {

		logger.Info("Начало выполнения StartPeriodSendingToGoogleSheet")

		// Получаем текущие время
		now := time.Now()
		duration := time.Minute * time.Duration(config.File.CacheConfig.ReplaceFormLiveTime)
		cutoffTime := now.Add(-duration)
		logger.Debug("Вычислен cutoffTime: ", cutoffTime.Format(time.RFC3339))

		// Получаем формы, которые не удалены, GoogleSheetStatus == false и CreatedAt старее 12 минут
		form, err := db.GetFirstFormForGoogleSheet(false, cutoffTime)
		if err != nil {
			logger.Error("Ошибка при получении форм для Google Sheets: ", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		logger.Info("Получена форма для отправки в Google Sheets: ", form)

		// Получаем последний индекс строки
		lastEmptyIndex, err := app.GetEmptyIndex(config.File.GoogleSheetConfig.ReplaceTableID, config.File.GoogleSheetConfig.ReplaceListName)
		if err != nil {
			logger.Error("Ошибка при получении пустого индекса строки: ", err)
			continue
		}
		logger.Info("Получен последний индекс строки: ", lastEmptyIndex)

		// Отправляем форму в Google Sheets
		err = app.SendToSheet(*form, lastEmptyIndex)
		if err != nil {
			logger.Error("Ошибка при отправке формы", form, " в Google Sheets: ", err)
			continue
		}

		logger.Info("Форма ID ", form.ID, " успешно отправлена в Google Sheets и статус обновлен")
	}
}

// Функция для генерации случайного ID
func GenerateID(idLength int) string {
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	var idCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, idLength)
	for i := range b {
		b[i] = idCharset[seededRand.Intn(len(idCharset))]
	}
	return string(b)
}

func (app *WebApp) PrepareTgMsg(form model.Form, date string) string {
	toTgString := ""
	ModuleLesson := "М" + form.Module + "У" + form.Lesson
	if form.ReplaceTransfer == "replace" {
		toTgString = "#" + form.ReplaceFormat + "\n" +
			form.Teacher + "\n" +
			"Тимлидер: " + form.TeamLeader + "\n" +
			"Дата и время урока: " + date + "  " + form.LessonTime + "\n" +
			"Номер группы: " + form.GroupNumber + "\n"

		toTgString += "Ученики (активные):" + fmt.Sprint(form.ActiveMembers) + "\n"

		if form.Subject == "НАСТАВНИЧЕСТВО" {
			toTgString += "Предмет: " + form.Subject + " " + config.File.WebConfig.MentoringManager + "\n"
		} else {
			toTgString += "Предмет: " + form.Subject + "\n"
		}

		if form.Module != "" || form.Lesson != "" {
			toTgString += "Номер урока: " + ModuleLesson + "\n"
		}

		if form.Link != "" {
			toTgString += "Cсылка на методический материал по уроку: " + form.Link + "\n"
		}

		// Если предмет, наставничество, значит важная информация состоит из 3 составляющих.
		if form.Subject == "НАСТАВНИЧЕСТВО" {
			impInf := form.MentoringInf1 + "\n" + form.MentoringInf2 + "\n" + form.MentoringInf3
			toTgString += "Важная информация: " + impInf
		} else {
			toTgString += "Важная информация: " + form.ImpInfo + "\n" + form.ImpInfo2
		}

		// "Причина: " + form.Reason + "\n"
	} else {
		toTgString = "#" + form.ReplaceFormat + "\n" +
			form.Teacher + "\n" +
			"Тимлидер: " + form.TeamLeader + "\n" +
			"Дата и время урока: " + date + "  " + form.LessonTime + "\n" +
			"Время переноса: " + form.TransferTime + "\n" +
			"Номер группы: " + form.GroupNumber + "\n"

		toTgString += "Ученики (активные):" + fmt.Sprint(form.ActiveMembers) + "\n"

		toTgString += "Предмет: " + form.Subject + "\n"

		if form.Module != "" || form.Lesson != "" {
			toTgString += "Номер урока: " + ModuleLesson + "\n"
		}
		// "Причина: " + form.Reason + "\n"
	}
	return toTgString
}

func CheckEmergency(form model.Form, date string) (bool, error) {
	// Форма срочная если дата и время урока за 24 часа
	emergency, err := IsDateTimeWithin24Hours(date, form.LessonTime)
	if err != nil {
		return false, err
	}
	// НО если это срочный перенос или переключатель "Перенос", то это не надо
	if (form.ReplaceFormat == "Срочный перенос") || (form.ReplaceTransfer == "transfer") || (form.ReplaceFormat == "Перенос на ближайшее время") || (form.ReplaceFormat == "Постоянный перенос") || (form.ReplaceFormat == "Перенос на неделю") {
		emergency = false
	}
	return emergency, nil
}

func IsDateTimeWithin24Hours(dateStr, timeStr string) (bool, error) {
	// Формат даты и времени
	layoutDate := "02.01.2006"
	layoutTime := "15:04"

	// Получаем текущую дату и время
	now := time.Now()

	// Парсим строку даты
	dateTimeStr := dateStr + " " + timeStr
	dateTime, err := time.Parse(layoutDate+" "+layoutTime, dateTimeStr)
	if err != nil {
		return false, err
	}

	// Добавляем 24 часа к текущему времени
	twentyFourHoursLater := now.Add(24 * time.Hour)

	// Проверяем условие
	return dateTime.Before(twentyFourHoursLater), nil
}

// getValidatedData извлекает данные для валидации пользователя, проверяет их и возвращает в случае успеха
func getValidatedData(initData string, token string) (*initdata.InitData, error) {
	if initData == "" {
		return nil, errors.New("missing parameter: initData")
	}
	initDataStruct, err := validateInitData(initData, token)
	if err != nil {
		return nil, err
	}

	return initDataStruct, nil
}

func validateInitData(initDataStr, token string) (*initdata.InitData, error) {
	expIn := 1 * time.Hour
	err := initdata.Validate(initDataStr, token, expIn)
	if err != nil {
		return nil, err
	}
	initData, err := initdata.Parse(initDataStr)
	if err != nil {
		return nil, err
	}
	return &initData, nil
}

func extractNumbers(s string) ([]uint64, error) {
	// Регулярное выражение для поиска всех чисел в строке
	re := regexp.MustCompile(`[0-9]+`)

	// Поиск всех чисел в строке
	matches := re.FindAllString(s, -1)

	var numbers []uint64
	for _, match := range matches {
		// Преобразование найденных строк в uint64
		num, err := strconv.ParseUint(match, 10, 64)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, num)
	}

	return numbers, nil
}
