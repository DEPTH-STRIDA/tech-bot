package clickhouse

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/pkg/clickhouse"
	"time"
)

var ClickHouseApp *clickhouse.ClickHouse = &clickhouse.ClickHouse{}

func init() {
	var err error

	ClickHouseApp, err = clickhouse.NewClickHouse(clickhouse.Config{
		DBHost:                   config.File.ClickHouseConfig.DBHost,
		DBName:                   config.File.ClickHouseConfig.DBName,
		DBUser:                   config.File.ClickHouseConfig.DBUser,
		DBPort:                   config.File.ClickHouseConfig.DBPort,
		DBPass:                   config.File.ClickHouseConfig.DBPass,
		DBNumberRepetitions:      config.File.ClickHouseConfig.DBNumberRepetitions,
		DBFormTableName:          config.File.ClickHouseConfig.DBFormTableName,
		CHPauseAfterSQLExecute:   config.File.ClickHouseConfig.CHPauseAfterSQLExecute,
		PauseAfterFailConnection: config.File.ClickHouseConfig.PauseAfterFailConnection,
		Logger:                   logger.Log,
		DB:                       db.DB,
	})

	if err != nil {
		logger.Log.Error("ClickHouseApp init error", err)
		panic(err)
	}

}

// ProcessRequests обрабатывает заявки и передает их в InsertRequests
func ProcessRequests(app *clickhouse.ClickHouse) {
	for {
		forms, err := app.FetchPendingRequests()
		if err != nil {
			logger.Log.Error("Ошибка извлечения заявок: %v", err)
			return
		}

		if len(forms) == 0 {
			logger.Log.Info("Нет заявок для обработки")
			return
		}

		err = app.InsertRequests(forms)
		if err != nil {
			logger.Log.Error("Ошибка вставки заявок: %v", err)
			return
		}

		// Обновляем статус заявок после успешной вставки
		err = app.UpdateRequestStatus(forms)
		if err != nil {
			logger.Log.Error("Ошибка обновления статуса заявок: %v", err)
			return
		}

		logger.Log.Info("Успешно обработаны и обновлены заявки")
	}
}

// ScheduleRequestProcessing запускает процесс обработки заявок в полночь и повторяет его ежедневно.
func ScheduleRequestProcessing(app *clickhouse.ClickHouse) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		duration := next.Sub(now)
		time.Sleep(duration)

		logger.Log.Info("Начало обработки заявок в полночь")
		ProcessRequests(app)
	}
}
