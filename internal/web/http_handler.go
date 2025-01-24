package web

import (
	"easycodeapp/internal/cache"
	"easycodeapp/internal/config"
	"easycodeapp/internal/googlesheet"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/notification"
	"easycodeapp/internal/utils"
	"easycodeapp/pkg/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	initData "github.com/telegram-mini-apps/init-data-golang"
)

type iniDataAPI struct {
	InitData string `json:"initData"`
}
type LoginDataAPI struct {
	iniDataAPI
	Email    string `json:"email"`
	Password string `json:"password"`
}
type SetDataRequest struct {
	model.Form
	InitDataString string `json:"initData"`
	InitData       *initData.InitData
}

// HandlePostData обработчки отправки данных
func (app *WebApp) HandlePostSetData(w http.ResponseWriter, r *http.Request) {
	// Парсинг тела запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("HandlePostDeleteData. Не удалось прочитать тело: ", err)
		http.Error(w, "HandlePostDeleteData. Не удалось прочитать тело: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Анмаршалинг формы
	var form SetDataRequest
	if err := json.Unmarshal(bodyBytes, &form); err != nil {
		logger.Error("HandlePostDeleteData. Ошибка при разборе JSON: ", err)
		http.Error(w, "HandlePostDeleteData. Ошибка при разборе JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Валидация и анмаршалинг телеграмм данных
	initData, err := getValidatedData(form.InitDataString, config.File.TelegramConfig.Token)
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandleGetHistoryData. Неверные телеграмм данные:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Установка полю данных тг
	form.InitData = initData
	form.TelegramUserID = initData.User.ID

	go app.SetData(form.Form, initData)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Форма успешно отправлена\nЕсли заявка не появится в беседе через 10 минут, то отправьте форму снова.")
}

// HandlePostDataEdit редактирования данных
func (app *WebApp) HandlePostEditData(w http.ResponseWriter, r *http.Request) {
	// Парсинг тела запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("HandlePostDataEdit. Не удалось прочитать тело: ", err)
		http.Error(w, "HandlePostDataEdit. Не удалось прочитать тело: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Анмашралинг формы
	var newForm SetDataRequest
	if err := json.Unmarshal(bodyBytes, &newForm); err != nil {
		logger.Error("HandlePostDataEdit. Ошибка при разборе JSON: ", err)
		http.Error(w, "HandlePostDataEdit. Ошибка при разборе JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Валидация и анмаршалинг телеграмм данных
	initData, err := getValidatedData(newForm.InitDataString, config.File.TelegramConfig.Token)
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandlePostDataEdit. Неверные телеграмм данные:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Получение старой заявки из базы данных
	oldForm, err := db.GetForm(int64(newForm.ID), time.Minute*time.Duration(config.File.CacheConfig.ReplaceFormLiveTime))
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandlePostDataEdit. Не удалось получить заявку" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Установка телеграмм данных
	newForm.InitData = initData
	newForm.TelegramUserID = initData.User.ID

	go app.EditData(newForm.Form, *oldForm, initData)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Успешно. Заявка скоро изменится.")
	logger.Info("(", r.RemoteAddr, ") HandlePostEditData Пользователь ", initData.User.Username, "(", initData.User.ID, ") отправил заявку на редактирование. ")
}

// HandlePostDeleteData обработчик удаления данных. Принимает заявку на удаление, проверяет данные и отвечает пользователю.
func (app *WebApp) HandlePostDeleteData(w http.ResponseWriter, r *http.Request) {
	// Парсинг тела запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("HandlePostDeleteData. Не удалось прочитать тело: ", err)
		http.Error(w, "HandlePostDeleteData. Не удалось прочитать тело: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Анмаршалинг формы
	var form model.DeleteForm
	if err := json.Unmarshal(bodyBytes, &form); err != nil {
		logger.Error("HandlePostDeleteData. Ошибка при разборе JSON: ", err)
		http.Error(w, "HandlePostDeleteData. Ошибка при разборе JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Валидация телеграмм данных
	initData, err := getValidatedData(form.InitData, config.File.TelegramConfig.Token)
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandleGetHistoryData. Неверные телеграмм данные:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	oldForm, err := db.GetForm(form.ID, time.Minute*time.Duration(config.File.CacheConfig.ReplaceFormLiveTime))
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandlePostDataEdit. Не удалось получить заявку" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go app.DeleteData(*oldForm, initData)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Заявка отправлена. Ожидайте.")
	logger.Info("(", r.RemoteAddr, ") HandlePostDeleteData Пользователь ", initData.User.Username, "(", initData.User.ID, ") отправил заявку на удаление. ")
}

type HistoryForm struct {
	model.Form
	CreationDate  string `json:"creation-date"`
	CreationTime  string `json:"creation-time"`
	RemainingTime string `json:"remaining-time"`
}

// HandleGetHistoryData обработчик получения истории
func (app *WebApp) HandleGetHistoryData(w http.ResponseWriter, r *http.Request) {
	// Валидация телеграмм данных
	initData, err := getValidatedData(r.URL.Query().Get("initData"), config.File.TelegramConfig.Token)
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandleGetHistoryData. Неверные телеграмм данные:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	forms, err := db.GetActiveUserForms(initData.User.ID, time.Minute*time.Duration(config.File.CacheConfig.ReplaceFormLiveTime))
	if err != nil {
		logger.Warn("(", r.RemoteAddr, ") HandleGetHistoryData. ", initData.User.Username, "(", initData.User.ID, ") не имеет заявок в кеше.")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "")
		return
	}

	// Преобразуем каждую форму в HistoryForm
	var historyForms []HistoryForm
	for _, form := range forms {
		historyForm := HistoryForm{
			Form:          form,
			CreationDate:  form.CreatedAt.Format("2006-01-02"),                                                                                  // Формат YYYY-MM-DD
			CreationTime:  form.CreatedAt.Format("15:04"),                                                                                       // Формат HH:MM
			RemainingTime: utils.CalculateRemainingTime(form.CreatedAt, time.Minute*time.Duration(config.File.CacheConfig.ReplaceFormLiveTime)), // Функция для расчёта оставшегося времени
		}
		historyForms = append(historyForms, historyForm)
	}

	jsonData, err := json.Marshal(historyForms)
	if err != nil {
		logger.Error("(", r.RemoteAddr, ") HandleGetHistoryData. ", initData.User.Username, "(", initData.User.ID, ") не удалось выполнить маршалинг json")
		http.Error(w, "Не удалось выполнить маршалинг json", http.StatusInternalServerError)
		return
	}
	jsonString := string(jsonData)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, jsonString)
	logger.Info("(", r.RemoteAddr, ") HandleGetHistoryData Пользователю ", initData.User.Username, "(", initData.User.ID, ") отправлены заявки. ")
	logger.Debug("(", r.RemoteAddr, ") HandleGetHistoryData Пользователю ", initData.User.Username, "(", initData.User.ID, ") отправлены заявки:\n", jsonString)
}

// HandleGetSelectData обработчик получения select данных
func (app *WebApp) HandleGetSelectData(w http.ResponseWriter, r *http.Request) {
	_, err := getValidatedData(r.URL.Query().Get("initData"), config.File.TelegramConfig.Token)
	if err != nil {
		logger.Warn("(" + r.RemoteAddr + ") Вход запрещен.HandleGetSelectData. Неверные телеграмм данные:" + err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Преобразуем структуру Data в JSON
	jsonData, err := json.Marshal(cache.GoogleSheetCacheApp.GetSelectData())
	if err != nil {
		logger.Error("(" + r.RemoteAddr + ") HandleGetSelectData. Ошибка во время маршалинга JSON (данные выпадающих списков): " + err.Error())
		http.Error(w, "Ошибка во время маршалинга JSON (данные выпадающих списков): "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовок Content-Type и отправляем JSON в ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
	logger.Info("(" + r.RemoteAddr + ") HandleGetSelectData. Успешный запрос данных выпадающих списков.")
}

const (
	updateSelectData        = "updateSelectData"
	dropService             = "dropService-disabled"
	sendMorningNotification = "sendMorningNotification"
)

// HandleGetHistoryData обработчик получения истории
func (app *WebApp) HandleInternalAdmin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		logger.Warn("Неправильный токен.")
		http.Error(w, "Неправильный токен.", http.StatusBadRequest)
		return
	}
	logger.Info("token: ", token)
	if token != "WXZq1BR4VUd40!ZecpKR?B4N1j0FyB33C-xGGJhaFaMbGe!E6oYm-wPCpFCjmKsH" {
		logger.Warn("Неправильный токен.")
		http.Error(w, "Неправильный токен.", http.StatusBadRequest)
		return
	}
	comand := r.URL.Query().Get("command")
	logger.Info("command: ", comand)
	if comand == "" {
		logger.Warn("Несуществующая команда.")
		http.Error(w, "Несуществующая команда.", http.StatusBadRequest)
		return
	}

	if comand == updateSelectData {
		err := googlesheet.ColectSelectData(googlesheet.GoogleSheet)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, errors.New("Ошибка при сборе данных select: "+err.Error()))
			logger.Info("Ошибка при сборе данных select: ", err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Успешно обновились select данные\n")
			logger.Info("Успешно обновились select данные.")
		}

	} else if comand == dropService {
		logger.Info("Админ запустил принудительный выход")
		os.Exit(1) // Завершаем программу с кодом выхода 1
	} else if comand == sendMorningNotification {
		notification.NotificationApp.StartHandleMorningMessage()
	}
}

func (app *WebApp) HandleReplace(w http.ResponseWriter, r *http.Request) {
	// Генерируем версию или используем существующую
	version := fmt.Sprintf("%d", time.Now().Unix())

	data := map[string]interface{}{
		"Version": version,
	}

	// рендер шаблона с данными
	err := app.render(w, "replace.page.tmpl", data)
	if err != nil {
		logger.Error("HandleReplace. Не удалось выполнить рендер: ", err)
		http.Error(w, "HandleReplace. Не удалось выполнить рендер: "+err.Error(), http.StatusInternalServerError)
	}
}

func (app *WebApp) HandleInternalAdminMenu(w http.ResponseWriter, r *http.Request) {
	// Генерируем версию или используем существующую
	version := fmt.Sprintf("%d", time.Now().Unix())

	data := map[string]interface{}{
		"Version": version,
	}

	// рендер шаблона с данными
	err := app.render(w, "admin-menu.page.tmpl", data)
	if err != nil {
		logger.Error("HandleInternalAdminMenu. Не удалось выполнить рендер: ", err)
		http.Error(w, "HandleInternalAdminMenu. Не удалось выполнить рендер: "+err.Error(), http.StatusInternalServerError)
	}
}

// HandleHome обработчик главной страницы. рендерит содержимое.
func (app *WebApp) HandleLogin(w http.ResponseWriter, r *http.Request) {
	// рендер шаблона
	err := app.render(w, "login.page.tmpl", map[string]interface{}{})
	if err != nil {
		logger.Error("HandleLogin. Не удалось выполнить рендер: ", err)
		http.Error(w, "HandleLogin. Не удалось выполнить рендер: "+err.Error(), http.StatusInternalServerError)
	}
}

// HandleHome обработчик главной страницы. рендерит содержимое.
func (app *WebApp) HandleHome(w http.ResponseWriter, r *http.Request) {
	// рендер шаблона
	err := app.render(w, "menu.page.tmpl", map[string]interface{}{})
	if err != nil {
		logger.Error("HandleHome. Не удалось выполнить рендер: ", err)
		http.Error(w, "HandleHome. Не удалось выполнить рендер: "+err.Error(), http.StatusInternalServerError)
	}
}
