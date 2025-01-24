package googlesheet

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/pkg/googlesheet"
	"sync"
	"time"
)

var GoogleSheet *googlesheet.GoogleSheets = &googlesheet.GoogleSheets{}

func init() {
	var err error

	GoogleSheet, err = googlesheet.NewGoogleSheets(googlesheet.Config{
		CredentialsFile:    config.File.GoogleSheetConfig.CredentialsFile,
		RequestUpdatePause: config.File.GoogleSheetConfig.RequestUpdatePause,
		Logger:             logger.Log,
	})
	if err != nil {
		logger.Error("Ошибка при создании GoogleSheets", err)
		return
	}

	if !config.File.WebConfig.IsTestMode {
		go StartPeriodUpdateCache()
	} else {
		logger.Info("ВКЛЮЧЕН ТЕСТОВЫЙ ЗАПУСК. Переодическое обновление данных из гугл таблиц отключено")
	}

	logger.Info("GoogleSheet успешно создан")
}

func StartPeriodUpdateCache() {
	var err error

	// Запускаем переодическое обновление вместе с получением
	time.Sleep(5 * time.Second)
	var wg sync.WaitGroup
	wg.Add(2)
	go startUpdateSelectData(GoogleSheet, &wg, &err)
	go startUpdateAdminsData(GoogleSheet, &wg, &err)

	if err != nil {
		logger.Error("Ошибка при запуске переодического обновления данных из гугл таблиц", err)
		return
	}

	logger.Info("Переодическое обновление данных из гугл таблиц запущено")
}
