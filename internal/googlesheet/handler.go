package googlesheet

import (
	"easycodeapp/internal/cache"
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/utils"
	"easycodeapp/pkg/googlesheet"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SetNewLine синхронно добавляет в строку заявку
// Если lineNumber <= 1, то функция самостоятельно ищет свободную строку.
// data - вставляемые данные, columns - буквы столбцов, куда вставляются данные
func SetNewForm(app *googlesheet.GoogleSheets, lineNumber int, data, columns []string, emergency bool) (int, error) {
	logger.Info("Начало выполнения SetNewForm. Номер строки:", lineNumber)
	conf := config.File.GoogleSheetConfig
	//Получение индекса последней свободной строки
	var lastEmptyIndex int
	var err error
	// Если номер приходит, то используем его и данные
	if lineNumber > 1 {
		lastEmptyIndex = lineNumber
	} else {
		// Если номер не приходит, то сами ищим
		lastEmptyIndex, err = app.GetEmptyIndex(conf.ReplaceTableID, conf.ReplaceListName, []string{"A", "B", "C"})
		if err != nil {
			return 0, err
		}
	}
	err = app.SetLine(conf.ReplaceTableID, conf.ReplaceListName, "A", "L", data, 0, lastEmptyIndex)
	if err != nil {
		return 0, err
	}

	// Галочка на срочность
	if emergency {
		err = app.SetLine(conf.ReplaceTableID, conf.ReplaceListName, conf.EmergencyCellName, conf.EmergencyCellName, []string{"true"}, 0, lastEmptyIndex)
	} else {
		err = app.SetLine(conf.ReplaceTableID, conf.ReplaceListName, conf.EmergencyCellName, conf.EmergencyCellName, []string{"false"}, 0, lastEmptyIndex)
	}
	if err != nil {
		return 0, err
	}

	logger.Info("Данные успешно вставлены в таблицу (номер строки = ", lastEmptyIndex, ")")
	logger.Info("Завершение SetNewForm. Вставлена строка:", lastEmptyIndex)
	return lastEmptyIndex, nil
}

// DeleteForm синхронно заполняет указанную строку пустыми значениями
// Если emergency = true, то снимает галочку срочности заявки
func DeleteForm(app *googlesheet.GoogleSheets, lineNumber int, emergency bool) error {
	logger.Info("Начало выполнения DeleteForm для строки:", lineNumber)
	conf := config.File.GoogleSheetConfig
	err := app.SetLine(conf.ReplaceTableID, conf.ReplaceListName, "A", "L", make([]string, 12), 0, lineNumber)
	if err != nil {
		return err
	}

	if emergency {
		err = app.SetLine(conf.ReplaceTableID, conf.ReplaceListName, conf.EmergencyCellName, conf.EmergencyCellName, []string{"false"}, 0, lineNumber)
		if err != nil {
			return err
		}
	}
	logger.Info("Завершение DeleteForm для строки:", lineNumber)
	return nil
}

// startUpdateSelectData переодически обновляет структуру select (данные выпадающих список)
func startUpdateSelectData(app *googlesheet.GoogleSheets, wg *sync.WaitGroup, err *error) {
	logger.Info("Запуск периодического обновления select данных")
	newErr := ColectSelectData(app)
	wg.Done()
	if newErr != nil {
		*err = newErr
		return
	}

	conf := config.File.GoogleSheetConfig
	for {
		time.Sleep(time.Hour * time.Duration(conf.SelectDataUpdatePauseHour))
		ColectSelectData(app)
	}
}

// ColectSelectData синхронная функция, которая выполняет запрос к таблице и обновляет данные select
func ColectSelectData(app *googlesheet.GoogleSheets) error {
	conf := config.File.GoogleSheetConfig
	logger.Info("Обновление данных выпадающих списков")

	cache.GoogleSheetCacheApp.Lock()
	defer cache.GoogleSheetCacheApp.Unlock()

	var err error

	cache.GoogleSheetCacheApp.Teachers, err = app.ReadColumnValues(conf.SelectDataTableID, conf.SelectDataListName, conf.SelectDataColumNames[0])
	if err != nil {
		return err
	}
	fmt.Println("Собраны преподавтели: ", cache.GoogleSheetCacheApp.Teachers)
	logger.Info("Получено преподавателей:", len(cache.GoogleSheetCacheApp.Teachers))

	cache.GoogleSheetCacheApp.Objects, err = app.ReadColumnValues(conf.SelectDataTableID, conf.SelectDataListName, conf.SelectDataColumNames[1])
	if err != nil {
		return err
	}
	logger.Info("Получено объектов:", len(cache.GoogleSheetCacheApp.Objects))

	cache.GoogleSheetCacheApp.ReplacementFormats, err = app.ReadColumnValues(conf.SelectDataTableID, conf.SelectDataListName, conf.SelectDataColumNames[2])
	if err != nil {
		return err
	}

	cache.GoogleSheetCacheApp.TransfermentFormats, err = app.ReadColumnValues(conf.SelectDataTableID, conf.SelectDataListName, conf.SelectDataColumNames[3])
	if err != nil {
		return err
	}

	cache.GoogleSheetCacheApp.TeamLeaders, err = app.ReadColumnValues(conf.SelectDataTableID, conf.SelectDataListName, conf.SelectDataColumNames[4])
	if err != nil {
		return err
	}

	logger.Info("Собраны новые select данные")
	logger.Info("Завершение сбора select данных")
	return err
}

// startUpdateAdminsData переодически обновляет структуру Admins (данные меню админов
func startUpdateAdminsData(app *googlesheet.GoogleSheets, wg *sync.WaitGroup, err *error) {
	newErr := ColectAdminsData(app)
	if newErr != nil {
		logger.Error("Не удалось собрать данные админов: ", newErr)
		*err = newErr
		return
	}
	wg.Done()

	conf := config.File.GoogleSheetConfig
	for {
		time.Sleep(time.Hour * time.Duration(conf.SelectDataUpdatePauseHour))
		ColectAdminsData(app)
	}
}

// ColectAdminsData синхронная функция, которая выполняет запрос к таблице и обновляет данные select
func ColectAdminsData(app *googlesheet.GoogleSheets) error {
	conf := config.File.GoogleSheetConfig
	listName := conf.AdminDataListName
	tableID := conf.SelectDataTableID
	logger.Info("Обновление данных выпадающих списков")

	cache.TelegramCacheApp.Lock()
	defer cache.TelegramCacheApp.Unlock()

	var err error
	var data [][]string

	// Получение максимальной ширины из ячейки K1
	maxWidthStr, err := app.GetCellValue(tableID, listName, "D1")
	if err != nil {
		return err
	}
	maxWidth, err := strconv.ParseInt(maxWidthStr, 10, 32)
	if err != nil {
		return err
	}
	logger.Info("Максимальная ширина:", maxWidth)

	// Получение последней строки из ячейки B1
	lastRowStr, err := app.GetCellValue(tableID, listName, "B1")
	if err != nil {
		return err
	}
	lastRow, err := strconv.ParseInt(lastRowStr, 10, 32)
	if err != nil {
		return err
	}
	logger.Info("Последняя строка:", lastRow)

	// Получение данных из матрицы
	data, err = app.GetMatrix(tableID, listName, 1, int(maxWidth), 2, int(lastRow))
	if err != nil {
		return err
	}
	logger.Info("Получение матрицы данных размером:", maxWidth, "x", lastRow)

	// Обход админов
	var Admins []int64
	for _, v := range data[0] {
		if strings.TrimSpace(v) != "" || v != "Админы ТГ id" {
			admin, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				logger.Error("Не удалось получить ID админа из строки: ", err)
				continue
			}
			Admins = append(Admins, admin)
			logger.Info("Админ ", admin, " добавлен в список админов")
		}
	}
	logger.Info("Обработка данных админов. Размер матрицы:", len(data), "x", len(data[0]))
	logger.Info("Получено админов:", len(Admins))

	// Обход кагорт
	var Cohorts [][]string = make([][]string, len(data))
	for i := 0; i < len(data); i++ { // Начинаем с 0, чтобы включить первую строку
		Cohorts[i] = make([]string, 0)

		logger.Debug("Собраны данные админов: ", (data[i]))

		for j := 0; j < len(data[i]); j++ {
			if strings.TrimSpace(data[i][j]) != "" {
				// Добавляем только значение из текущей ячейки
				Cohorts[i] = append(Cohorts[i],
					strings.TrimSpace(data[i][j]))
			}
		}
	}
	logger.Info("Получено когорт:", len(Cohorts))

	cache.TelegramCacheApp.TgAdminIDS = Admins
	cache.TelegramCacheApp.Cohorts = Cohorts

	lastRowStr, err = app.GetCellValue(tableID, listName, "K1")
	if err != nil {
		return err
	}
	chatIdsStr, err := app.GetColumnValues(tableID, listName, "K3:K"+lastRowStr)
	if err != nil {
		return err
	}
	var teamChats []int64

	for _, v := range chatIdsStr {
		teamChat, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			logger.Error("Не удалось получить ID командного чата из строки: ", err)
			continue
		}
		teamChats = append(teamChats, teamChat)
	}
	logger.Info("Получено командных чатов:", len(teamChats))
	cache.TelegramCacheApp.TeamChats = teamChats

	logger.Info("Собраны новые admin данные")

	return err
}

type BirthDay struct {
	Name       string // 1
	UserName   string // 4
	Experience string // Фактически нет, но дата старта работы: 14
	Date       string // 15
}

func CollectBirthDays(app *googlesheet.GoogleSheets) (*[]BirthDay, error) {
	conf := config.File.GoogleSheetConfig

	logger.Info("Сбор данных о ДР: ", conf.BirthDayDataListName, " в таблице: ", conf.BirthDayDataTableID)

	// Получение последней свободной строки
	emptyIndex, err := app.GetEmptyIndex(conf.BirthDayDataTableID, conf.BirthDayDataListName, []string{"B", "C"})
	if err != nil {
		return nil, err
	}
	logger.Info("Найдена последняя строка:", emptyIndex)

	// Получени инфы о всех преподавателях.  A -> P
	users, err := app.GetMatrix(conf.BirthDayDataTableID, conf.BirthDayDataListName, 1, 16, 1, emptyIndex)
	if err != nil {
		return nil, err
	}
	logger.Info("Получено строк данных:", len(users))
	logger.Debug("Собрано ", len(users), " строк инфы о преподавателях. Инфа о ДР.")

	var birthDays []BirthDay

	for i := 0; i < len(users[0]); i++ {
		// Проверка на пустые поля
		if users[1][i] == "" {
			logger.Debug("Пустое значение в поле 'Имя' в строке ", i)
			continue
		}
		if users[4][i] == "" {
			logger.Debug("Пустое значение в поле 'Пользователь' в строке ", i)
			continue
		}
		if users[14][i] == "" {
			logger.Debug("Пустое значение в поле 'Дата начала работы' в строке ", i)
			continue
		}
		if users[15][i] == "" {
			logger.Debug("Пустое значение в поле 'Дата рождения' в строке ", i)
			continue
		}
		// logger.Debug("Обработка преподавателя: ", users[1][i], " с датой рождения: ", users[15][i])

		// Парсинг даты ДР
		var startDate time.Time
		startDate, err = utils.ParseDate(users[14][i])
		if err != nil {
			logger.Error("Ошибка парсинга даты:", err)
			continue
		}

		// Парсинг даты старта раб��ты
		var birthDate time.Time
		birthDate, err = utils.ParseDate(users[15][i])
		if err != nil {
			logger.Error("Ошибка парсинга даты:", err)
			continue
		}

		isDay := time.Now().Day() == birthDate.Day()
		isMonth := time.Now().Month() == birthDate.Month()

		if !isDay || !isMonth {
			continue
		}

		// Текущая дата
		now := time.Now()

		// Вычисление разницы
		years, months, days := utils.CalculateDifference(startDate, now)

		// Получение инфы об опыте
		expireince := ""
		if years > 0 {
			expireince += fmt.Sprintf("%d лет, ", years)
		}
		if months > 0 {
			expireince += fmt.Sprintf("%d месяцев, ", months)
		}
		expireince += fmt.Sprintf("%d дней", days)

		user := BirthDay{
			Name:       users[1][i],
			UserName:   users[4][i],
			Date:       users[15][i],
			Experience: expireince,
		}

		birthDays = append(birthDays, user)
	}

	logger.Info("Завершение сбора данных о днях рождения")
	logger.Info("Найдено именинников:", len(birthDays))

	return &birthDays, nil
}
