package main

import (
	"os"

	"easycodeapp/internal/infrastructure/clickhouse"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/notification"
	"easycodeapp/internal/tg"
	u "easycodeapp/internal/utils"
	"easycodeapp/internal/web"
)

func main() {
	HandleFatalError(u.InitGlobalLocationTime())

	go clickhouse.ScheduleRequestProcessing(clickhouse.ClickHouseApp)

	HandleFatalError(tg.InitTelegramBot())

	HandleFatalError(tg.InitAdminApp())

	HandleFatalError(notification.InitNotificationApp())

	HandleFatalError(web.InitWebApp())

	HandleFatalError(web.App.HandleUpdates())
}

// HandleFatalError если err ошибка, то логгирует ее, отправляет всем админам в тг, если ошибки нет, то возвращает nil
func HandleFatalError(err error) error {
	if err != nil {
		logger.Error("Критическая ошибка: ", err)

		if tg.TelegramBot != nil {
			tg.TelegramBot.SendAllAdmins("Критическая ошибка: " + err.Error())
		}
		os.Exit(1)
	}
	return nil
}
