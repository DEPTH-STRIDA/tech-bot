package web

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/googlesheet"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/tg"
	"easycodeapp/pkg/model"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Отправляет данные в гугл таблицу.
func (app *WebApp) SendToSheet(form model.Form, lastEmptyIndex int) error {
	// logger.Info("Получение слайса данных, для отправки в таблицу. Индекс: ", lastEmptyIndex)
	// Получение слайса данных, для отправки в таблицу
	slice, err := ConvertFormDataToSlice(form)
	if err != nil {
		return err
	}

	// Дата заявки
	logger.Error("Дата заявки")
	date, err := ConvertDateFormat(form.LessonDate)
	if err != nil {
		logger.Error("Не удалось распарсить дату урока: " + err.Error())
		tg.TelegramBot.SendAllAdmins("Не удалось распарсить дату урока: " + err.Error())
	}
	// Срочность заявк
	emergency, err := CheckEmergency(form, date)
	if err != nil {
		logger.Error("Не удалось определить срочность заявки: " + err.Error())
		tg.TelegramBot.SendAllAdmins("Не удалось определить срочность заявки: " + err.Error())
	}

	for i := 0; i <= config.File.WebConfig.NumberRepetitions; i++ {
		logger.Debug("Попытка отправки в таблицу номер: ", i)

		index, err := googlesheet.SetNewForm(googlesheet.GoogleSheet, lastEmptyIndex, slice, config.File.GoogleSheetConfig.RequiredFreeFields, emergency)
		if err != nil {
			logger.Error("Ошибка при отправке данных в таблицу (попытка " + fmt.Sprint(i+1) + "): " + err.Error() + "\nНовая попытка...")
			time.Sleep(time.Duration(config.File.WebConfig.RepeatPause) * time.Second)
		} else {
			// Обновление записи в базе данных
			updatedFields := map[string]interface{}{
				"google_sheet_status":      true,
				"google_sheet_line_number": index,
			}
			db.UpdateFormByID(form.ID, updatedFields)
			return nil
		}
	}
	errMsg := "Ни одна попытка отправить форму в таблицу, так и не удалась:\n" + fmt.Sprint(form) + "\n" + err.Error()

	newMsg := tgbotapi.NewMessage(config.File.TelegramConfig.BotTgChat, errMsg)
	newMsg.ParseMode = "html"
	newMsg.ReplyToMessageID = config.File.TelegramConfig.ErrorTopicID

	tg.TelegramBot.SendMessageRepetLowPriority(newMsg, 3)

	logger.Info("Ни одна попытка отправить данные в таблицу, так и не удалась. Форма: ", form)
	tg.TelegramBot.SendAllAdmins("Ни одна попытка отправить данные в таблицу, так и не удалась. Форма: " + fmt.Sprint(form))
	return nil
}

var LastEmptyIndex int = 0

// GetEmptyIndex получает первый незанятый индекс строки в Google Sheets. Проверяет наличие записи с этим GoogleSheetLineNumber в БД.
func (app *WebApp) GetEmptyIndex(tableID, ListName string) (int, error) {
	// Поиск места
	for i := 0; i < config.File.WebConfig.NumberRepetitions; i++ {

		// Получаем пустой индекс строки
		lastEmptyIndex, err := googlesheet.GoogleSheet.GetEmptyIndex(tableID, ListName, []string{"A", "B", "C"})
		if err != nil {
			logger.Error("Не удалось получить пустой индекс строки, новая попытка: ", i+1, " ошибка: ", err)
			continue
		}
		return lastEmptyIndex, nil
		// // Проверка наличия записи с этим GoogleSheetLineNumber в БД
		// _, err = db.GetFormByGoogleSheetLineNumber(lastEmptyIndex)
		// // Если ошибка, значит записи с таким GoogleSheetLineNumber нет в БД
		// if err != nil {
		// 	logger.Info("Индекс ", lastEmptyIndex, " не занят, возвращаем его")
		// 	return lastEmptyIndex, nil
		// }

		// logger.Info("Индекс ", lastEmptyIndex, " уже занят, продолжаем поиск")

	}
	return 0, fmt.Errorf("не удалось получить незанятый индекс после %d попыток", config.File.WebConfig.NumberRepetitions)
}

func (app *WebApp) EditInSheet(f model.Form, lineNumber int, userID int64, UID string) {
	logger.Debug("Запуск функции редактировании в чате")
	slice, err := ConvertFormDataToSlice(f)
	if err != nil {
		logger.Error("Не удалось конвертироать данные в слайс: ", err)
		tg.TelegramBot.SendAllAdmins("Не удалось конвертироать данные в слайс: " + err.Error())
		return
	}

	for i := 1; i <= config.File.WebConfig.NumberRepetitions; i++ {
		logger.Debug("Попытка редактирования в таблице номер: ", i)

		index, err := googlesheet.SetNewForm(googlesheet.GoogleSheet, lineNumber, slice, config.File.GoogleSheetConfig.RequiredFreeFields, false)
		if err != nil {
			logger.Error("Ошибка при редактировании данных в таблице (попытка " + fmt.Sprint(i+1) + "): " + err.Error() + "\nНовая попытка...")
			time.Sleep(time.Duration(config.File.WebConfig.RepeatPause) * time.Second)
		} else {
			// Обновление записи в базе данных
			updatedFields := map[string]interface{}{
				"google_sheet_status":      true,
				"google_sheet_line_number": index,
			}
			app.updateDatabase(f, updatedFields, fmt.Sprint(f))
			return
		}
	}

	logger.Info("Ни одна попытка редактировать данные в таблице, так и не удалась. Форма: ", f)
	tg.TelegramBot.SendAllAdmins("Ни одна попытка редактировать данные в таблице, так и не удалась. Форма: " + fmt.Sprint(f))
}

func (app *WebApp) DeleteInSheet(toEditDataForm model.Form, userID int64, UID string, emergency bool) {
	logger.Debug("Запуск функции удаления из таблицы")

	for i := 1; i <= config.File.WebConfig.NumberRepetitions; i++ {
		lineNumber := toEditDataForm.GoogleSheetLineNumber

		err := googlesheet.DeleteForm(googlesheet.GoogleSheet, lineNumber, emergency)
		if err != nil {
			logger.Error("Ошибка при удалении данных из таблицы (попытка " + fmt.Sprint(i+1) + "): " + err.Error() + "\nНовая попытка...")
			time.Sleep(time.Duration(config.File.WebConfig.RepeatPause) * time.Second)
		} else {
			// Обновление записи в базе данных
			updatedFields := map[string]interface{}{
				"google_sheet_status":      false,
				"google_sheet_line_number": 0,
			}
			app.updateDatabase(toEditDataForm, updatedFields, fmt.Sprint(toEditDataForm))
			return
		}
	}
	logger.Info("Ни одна попытка удалить данные из таблицы, так и не удалась. Форма: ", toEditDataForm)
	tg.TelegramBot.SendAllAdmins("Ни одна попытка удалить данные из таблицы, так и не удалась. Форма: " + fmt.Sprint(toEditDataForm))
}

// ConvertFormDataToSlice конфентирует заявку в слайс строк, для последующей отправки, например, в таблицы.
func ConvertFormDataToSlice(data model.Form) ([]string, error) {
	logger.Error("Дата занятия")
	// Дата занятия
	date, err := ConvertDateFormat(data.LessonDate)
	if err != nil {
		return nil, err
	}
	logger.Error("Дата отправки")
	// Дата отправки
	sendTime := data.CreatedAt.Format("02.01.2006 15:04")

	// Форматирование номеров модуля и урока
	ModuleLesson := "М" + data.Module + "У" + data.Lesson
	// Форматирование количества учеников в группах
	members := "Ученики (активные):" + fmt.Sprint(data.ActiveMembers)

	slice := []string{}
	slice = append(slice, sendTime)
	slice = append(slice, date)
	slice = append(slice, data.LessonTime)
	slice = append(slice, data.ReplaceFormat)
	slice = append(slice, data.GroupNumber)
	slice = append(slice, "https://school.easy-mo.ru/courses-view?course_id="+data.GroupNumber)
	slice = append(slice, data.Teacher)
	slice = append(slice, data.Subject)
	slice = append(slice, ModuleLesson+" "+data.Link)

	// Если предмет, наставничество, значит важная информация состоит из 3 составляющих.
	if data.Subject == "НАСТАВНИЧЕСТВО" {
		impInf := data.MentoringInf1 + "\n" + data.MentoringInf2 + "\n" + data.MentoringInf3
		slice = append(slice, data.TransferTime+" \n"+impInf)
	} else {
		slice = append(slice, data.TransferTime+" \n"+data.ImpInfo+"\n"+data.ImpInfo2)
	}

	slice = append(slice, members)
	slice = append(slice, data.Reason)
	return slice, nil
}
