package web

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/easycodeapi"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/tg"
	"easycodeapp/pkg/model"
	"encoding/json"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

func (app *WebApp) SetData(form model.Form, initData *initdata.InitData) {
	// Сериализация формы в JSON для логирования
	formJSON, err := json.Marshal(form)
	if err != nil {
		logger.Error("Не удалось сериализовать форму в JSON: ", err)
	} else {
		logger.Info("Пользователь ", initData.Chat.Username, " отправил заявку\n", string(formJSON))
	}

	// Получение номеров групп
	groupNumbers, err := extractNumbers(form.GroupNumber)
	if err != nil {
		logger.Error("Не удалось распарсить номера групп: ", form.GroupNumber)
		tg.TelegramBot.SendAllAdmins("Не удалось распарсить номера групп: " + form.GroupNumber)
	} else {
		// Получения количества учеников (для каждой группы)
		_, form.ActiveMembers, err = easycodeapi.Api.GetGroupsStats(groupNumbers)
		if err != nil {
			logger.Info("Не удалось получить количество учеников")
		}
	}

	// Дата заявки
	date, err := ConvertDateFormat(form.LessonDate)
	if err != nil {
		logger.Error("Не удалось распарсить дату урока: " + err.Error())
		tg.TelegramBot.SendAllAdmins("Не удалось распарсить дату урока: " + err.Error())
	}

	// Срочность заявки
	emergency, err := CheckEmergency(form, date)
	if err != nil {
		logger.Error("Не удалось определить срочность заявки: " + err.Error())
		tg.TelegramBot.SendAllAdmins("Не удалось определить срочность заявки: " + err.Error())
	}

	// Установка срочности заявки
	form.IsEmergency = emergency

	// Создание заявки в базе данных
	f, err := db.CreateFormIfNoSimilar(form, time.Minute*time.Duration(config.File.CacheConfig.ReplaceFormLiveTime))
	if err != nil {
		logger.Error("Не удалось распарсить номера групп: ", form.GroupNumber)
		tg.TelegramBot.SendAllAdmins("Не удалось создать запись заявки: " + err.Error())
		tg.TelegramBot.SendMessage(tgbotapi.NewMessage(initData.User.ID, "Не удалось создать заявку: "+err.Error()))
	}

	// Проверка на ошибку создания заявки
	if f == nil || f.ID == 0 {
		logger.Error("F Не удалось создать запись заявки: ", err)
		tg.TelegramBot.SendAllAdmins("F Не удалось создать запись заявки: " + err.Error())
		tg.TelegramBot.SendMessage(tgbotapi.NewMessage(initData.User.ID, "F Не удалось создать заявку: "+err.Error()))
	}

	// Установка ID заявки
	form.ID = f.ID

	// Отправляем данные в обычный чат телеграмм
	app.reqTelegram.HandleRequest(func() error {
		app.SendToChat(form, date, ReplaceChat, config.File.TelegramConfig.ReplaceChatID)
		return nil
	})
	// Отправляем данные в спецназ чат, если это надо
	if emergency {
		app.reqTelegram.HandleRequest(func() error {
			app.SendToChat(form, date, EmergencyChat, config.File.TelegramConfig.EmergencyChatID)
			return nil
		})
	}
}

const (
	// Если обе простые, но надо редактировать только в одном чате
	DefaultDefault = "DefaultDefault"
	// Если обе срочные, то надо редактировать в двух чатах
	EmergencyEmergency = "EmergencyEmergency"
	// Если была обычной, то надо отредактировать в старом чате, и отправить в срочный
	DefaultToEmergency = "DefaultToEmergency"
	// Если была срочной, то надо удалить из срочного чата, и отредактировать в обычном
	EmergencyToDefault = "EmergencyToDefault"
)

func (app *WebApp) EditData(newForm model.Form, oldForm model.Form, initData *initdata.InitData) {
	logger.Info("Начало выполнения EditData")
	logger.Debug("Входные параметры - newForm.ID: ", newForm.ID, ", oldForm.ID: ", oldForm.ID, ", initData.User.ID: ", initData.User.ID)

	// Конвертация даты урока
	newDate, err := ConvertDateFormat(newForm.LessonDate)
	if err != nil {
		logger.Error("Не удалось распарсить дату урока: ", err)
		tg.TelegramBot.SendAllAdmins("Не удалось распарсить дату урока: " + err.Error())
	} else {
		logger.Debug("Конвертированная дата урока: ", newDate)
	}

	// Извлечение номеров групп
	groupNumbers, err := extractNumbers(newForm.GroupNumber)
	if err != nil {
		logger.Info("Не удалось распарсить номера групп: ", newForm.GroupNumber)
	} else {
		logger.Debug("Извлеченные номера групп: ", groupNumbers)
		// Получение количества активных учеников для каждой группы
		count, activeMembers, err := easycodeapi.Api.GetGroupsStats(groupNumbers)
		if err != nil {
			logger.Info("Не удалось получить количество учеников: ", err)
		} else {
			newForm.ActiveMembers = activeMembers
			logger.Debug("Получено activeMembers: ", activeMembers, ", count: ", count)
		}
	}

	// Проверка статуса срочности формы после редактирования
	newEmergency, err := CheckEmergency(newForm, newDate)
	if err != nil {
		logger.Error("Не удалось определить срочность заявки: ", err)
		tg.TelegramBot.SendAllAdmins("Не удалось определить срочность заявки: " + err.Error())
	} else {
		logger.Debug("Определена новая срочность: ", newEmergency)
	}
	oldEmergency := oldForm.IsEmergency

	// Обновление срочности заявки в базе данных
	updatedFields := map[string]interface{}{
		"is_emergency":     newEmergency,
		"lesson_time":      newForm.LessonTime,
		"lesson_date":      newForm.LessonDate,
		"replace_format":   newForm.ReplaceFormat,
		"group_number":     newForm.GroupNumber,
		"teacher":          newForm.Teacher,
		"subject":          newForm.Subject,
		"module":           newForm.Module,
		"lesson":           newForm.Lesson,
		"reason":           newForm.Reason,
		"replace_transfer": newForm.ReplaceTransfer,
		"link":             newForm.Link,
		"imp_info":         newForm.ImpInfo,
		"imp_info2":        newForm.ImpInfo2,
		"mentoring_inf_1":  newForm.MentoringInf1,
		"mentoring_inf_2":  newForm.MentoringInf2,
		"mentoring_inf_3":  newForm.MentoringInf3,
		"transfer_time":    newForm.TransferTime,
		"team_leader":      newForm.TeamLeader,
	}
	db.UpdateFormByID(newForm.ID, updatedFields)

	// Подготовка сообщения для Telegram
	toTgString := app.PrepareTgMsg(newForm, newDate)
	logger.Debug("Подготовлено сообщение для Telegram: ", toTgString)

	// Редактирование сообщения в обычном чате, если требуется
	if oldForm.ReplaceTgStatus {
		logger.Info("Требуется редактировать сообщение в ReplaceChat с ReplaceMsgId: ", oldForm.ReplaceMsgId)
		err := app.reqTelegram.HandleRequest(func() error {
			app.EditInChat(oldForm, ReplaceChat, toTgString, config.File.TelegramConfig.ReplaceChatID, oldForm.ReplaceMsgId)
			return nil
		})
		if err != nil {
			logger.Error("Ошибка при редактировании сообщения в ReplaceChat: ", err)
		} else {
			logger.Info("Сообщение успешно отредактировано в ReplaceChat")
		}
	}

	// Логика обработки срочности
	if oldEmergency {
		logger.Debug("Старый статус срочности был true")
		if newEmergency {
			logger.Info("Статус срочности остался true. Редактирование сообщения в EmergencyChat с EmergencyMsgId: ", oldForm.EmergencyMsgId)
			err := app.reqTelegram.HandleRequest(func() error {
				app.EditInChat(oldForm, EmergencyChat, toTgString, config.File.TelegramConfig.EmergencyChatID, oldForm.EmergencyMsgId)
				return nil
			})
			if err != nil {
				logger.Error("Ошибка при редактировании сообщения в EmergencyChat: ", err)
			} else {
				logger.Info("Сообщение успешно отредактировано в EmergencyChat")
			}
		} else {
			logger.Info("Статус срочности изменился на false. Удаление сообщения из EmergencyChat с EmergencyMsgId: ", oldForm.EmergencyMsgId)
			err := app.reqTelegram.HandleRequest(func() error {
				app.DeleteInChat(oldForm, EmergencyChat, config.File.TelegramConfig.EmergencyChatID, oldForm.EmergencyMsgId)
				return nil
			})
			if err != nil {
				logger.Error("Ошибка при удалении сообщения из EmergencyChat: ", err)
			} else {
				logger.Info("Сообщение успешно удалено из EmergencyChat")
			}
		}
	} else {
		logger.Debug("Старый статус срочности был false")
		if newEmergency {
			logger.Info("Статус срочности изменился на true. Отправка сообщения в EmergencyChat")
			err := app.reqTelegram.HandleRequest(func() error {
				app.SendToChat(newForm, newDate, EmergencyChat, config.File.TelegramConfig.EmergencyChatID)
				return nil
			})
			if err != nil {
				logger.Error("Ошибка при отправке сообщения в EmergencyChat: ", err)
			} else {
				logger.Info("Сообщение успешно отправлено в EmergencyChat")
			}
		} else {
			logger.Debug("Статус срочности остался false. Никаких действий не требуется")
		}
	}

	logger.Info("Завершение выполнения EditData")
}

func (app *WebApp) DeleteData(toDeleteDataForm model.Form, initData *initdata.InitData) {

	// Логирование начала процесса удаления
	logger.Info("Начало удаления формы. Пользователь: ", initData.Chat.Username, ", ID формы: ", toDeleteDataForm.ID)

	// Сериализация данных формы для логирования
	bytes, err := json.Marshal(toDeleteDataForm)
	if err != nil {
		logger.Error("Не удалось сериализовать данные формы для удаления: ", err)
	} else {
		logger.Debug("Удаление формы с данными: ", string(bytes))
	}

	// Удаление из обычного чата
	if toDeleteDataForm.ReplaceTgStatus {
		logger.Info("Форма требует удаления из обычного чата. ID сообщения: ", toDeleteDataForm.ReplaceMsgId)
		app.DeleteInChat(toDeleteDataForm, ReplaceChat, config.File.TelegramConfig.ReplaceChatID, toDeleteDataForm.ReplaceMsgId)
	} else {
		logger.Debug("Форма не требует удаления из обычного чата.")
	}

	// Удаление из срочного чата
	if toDeleteDataForm.EmergencyTgStatus {
		logger.Info("Форма требует удаления из срочного чата. ID сообщения: ", toDeleteDataForm.EmergencyMsgId)
		app.DeleteInChat(toDeleteDataForm, EmergencyChat, config.File.TelegramConfig.EmergencyChatID, toDeleteDataForm.EmergencyMsgId)
	} else {
		logger.Debug("Форма не требует удаления из срочного чата.")
	}

	// Удаление из базы данных
	logger.Info("Попытка удаления формы из базы данных. ID формы: ", toDeleteDataForm.ID)
	err = db.DeleteForm(toDeleteDataForm.ID)
	if err != nil {
		logger.Error("Ошибка при удалении заявки из БД. ID формы: ", toDeleteDataForm.ID, ", Ошибка: ", err)
	} else {
		logger.Info("Успешно удалена форма из базы данных. ID формы: ", toDeleteDataForm.ID)
	}

	// Логирование завершения процесса удаления
	logger.Info("Завершено удаление формы. Пользователь: ", initData.Chat.Username, ", ID формы: ", toDeleteDataForm.ID)
}

func GetTransitionPlan(newForm, oldForm bool) string {
	switch {
	case oldForm && newForm:
		return EmergencyEmergency
	case !oldForm && !newForm:
		return DefaultDefault
	case oldForm && !newForm:
		return EmergencyToDefault
	case !oldForm && newForm:
		return DefaultToEmergency
	default:
		return EmergencyEmergency
	}
}

// getTimeByTimezone принимает временную зону и возвращает дату и время в этой зоне.
func GetTimeByTimezone(timezone string) (string, string, error) {
	// Загружаем временную зону
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", "", fmt.Errorf("не удалось загрузить временную зону: %v", err)
	}

	// Получаем текущее время в указанной временной зоне
	now := time.Now().In(loc)

	// Форматируем дату и время
	date := now.Format("2006-01-02")
	time := now.Format("15:04:05")

	return date, time, nil
}
