package clickhouse

import (
	"easycodeapp/pkg/model"
	"fmt"
	"strings"
	"time"
)

// InsertRequests вставляет массив заявок в ClickHouse. Если массив пуст, функция завершает выполнение без ошибок.
func (app *ClickHouse) InsertRequests(requests []model.Form) error {
	if len(requests) == 0 {
		return nil
	}
	return app.insertRequests(requests)
}

// insertRequests выполняет фактическую вставку заявок в ClickHouse, формируя SQL-запрос и отправляя его на выполнение.
func (app *ClickHouse) insertRequests(requests []model.Form) error {
	conf := app.config

	app.log("Начат процесс отправки заявок insertRequests")
	var values []string
	for _, req := range requests {
		value, err := app.formatRequest(req)
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	insertSQL := fmt.Sprintf("INSERT INTO `%s`.`%s` (id, form_date, form_time, lesson_date, lesson_time, replace_format, group_number, teacher, subject, module, lesson, reason, replace_transfer_format, link, imp_info, transfer_time, team_leader) VALUES %s",
		conf.DBName, conf.DBFormTableName, strings.Join(values, ", "))

	app.log("Был подготовлен запрос: " + insertSQL)
	resp, err := app.executeSQL(insertSQL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return app.handleResponse(resp)
}

// formatRequest форматирует одну заявку в строку SQL с экранированием значений для безопасной вставки в ClickHouse.
func (app *ClickHouse) formatRequest(req model.Form) (string, error) {
	// Функция для парсинга даты и времени с обработкой различных форматов
	parseDateTime := func(date, timeStr string) (time.Time, error) {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02 15:04:05.000000-07",
		}

		dateTimeStr := fmt.Sprintf("%s %s", date, timeStr)

		for _, format := range formats {
			if t, err := time.Parse(format, dateTimeStr); err == nil {
				return t, nil
			}
		}

		return time.Time{}, fmt.Errorf("не удалось распарсить дату и время: %s", dateTimeStr)
	}

	// Парсим дату и время создания формы
	creationDateTime, err := parseDateTime(req.CreatedAt.Format("2006-01-02"), req.CreatedAt.Format("15:04:05"))
	if err != nil {
		app.logError(fmt.Sprintf("Ошибка парсинга даты и времени создания: %v. CreationDate: %s, CreationTime: %s", err, req.CreatedAt.Format("2006-01-02"), req.CreatedAt.Format("15:04:05")))
		// Используем текущее время как запасной вариант
		creationDateTime = time.Now()
	}

	// Парсим дату и время урока
	lessonDateTime, err := parseDateTime(req.LessonDate, req.LessonTime)
	if err != nil {
		app.logError(fmt.Sprintf("Ошибка парсинга даты и времени урока: %v. LessonDate: %s, LessonTime: %s", err, req.LessonDate, req.LessonTime))
		// Используем время создания как запасной вариант
		lessonDateTime = creationDateTime
	}

	// Функция для экранирования строковых значений
	escape := func(value string) string {
		return strings.ReplaceAll(value, "'", "\\'")
	}

	// Форматируем значения в соответствии с типами колонок в ClickHouse
	return fmt.Sprintf("(0, '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s')",
		creationDateTime.Format("2006-01-02"),
		creationDateTime.Format("2006-01-02 15:04:05"),
		lessonDateTime.Format("2006-01-02"),
		lessonDateTime.Format("2006-01-02 15:04:05"),
		escape(req.ReplaceFormat),
		escape(req.GroupNumber),
		escape(req.Teacher),
		escape(req.Subject),
		escape(req.Module),
		escape(req.Lesson),
		escape(req.Reason),
		escape(req.ReplaceTransfer),
		escape(req.Link),
		escape(req.ImpInfo),
		escape(req.TransferTime),
		escape(req.TeamLeader)), nil
}

// FetchPendingRequests извлекает до 200 заявок из базы данных, у которых ClickhouseStatus = false.
func (app *ClickHouse) FetchPendingRequests() ([]model.Form, error) {
	var forms []model.Form
	err := app.db.Where("clickhouse_status = ?", false).Limit(200).Find(&forms).Error
	return forms, err
}

// UpdateRequestStatus обновляет статус заявок в базе данных после успешной вставки в ClickHouse.
func (app *ClickHouse) UpdateRequestStatus(requests []model.Form) error {
	for _, req := range requests {
		err := app.db.Model(&req).Update("clickhouse_status", true).Error
		if err != nil {
			app.logError("Ошибка обновления статуса заявки: %v", err)
			return err
		}
	}
	return nil
}
