package tg

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/db"
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/utils"
	"easycodeapp/pkg/model"
	"easycodeapp/pkg/request"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gocache "github.com/patrickmn/go-cache"
)

var TelegramBot *Bot

type Request func() error

type UserState struct {
	OldState     string
	CurrentState string
}

type Bot struct {
	botAPI *tgbotapi.BotAPI

	HandlCallbackLessonButton func(app *Bot, update tgbotapi.Update, UID string) error

	sessions   *SessionsCache
	userStates *gocache.Cache

	states map[string]State

	msgRequestHandler      *request.RequestHandler
	callbackRequestHandler *request.RequestHandler
}

func InitTelegramBot() error {
	var err error

	TelegramBot, err = NewBot(UserStates)
	if err != nil {
		return err
	}

	return nil
}

// Конструктор нового бота
func NewBot(states map[string]State) (*Bot, error) {
	if states == nil {
		return nil, fmt.Errorf("states shouldn't be nil")
	}

	conf := config.File.TelegramConfig

	msgRequestHandler, err := request.NewRequestHandler(request.Config{
		BufferSize: conf.MsgBufferSize,
		Logger:     logger.Log,
	})
	if err != nil {
		return nil, err
	}
	callbackRequestHandler, err := request.NewRequestHandler(request.Config{
		BufferSize: conf.CallBackBufferSize,
		Logger:     logger.Log,
	})
	if err != nil {
		return nil, err
	}

	app := Bot{
		msgRequestHandler:      msgRequestHandler,
		callbackRequestHandler: callbackRequestHandler,
		sessions:               NewSessionsCache(),
		userStates:             gocache.New(24*time.Hour, 30*time.Minute),
		states:                 states,
	}

	go app.msgRequestHandler.ProcessRequests(time.Duration(conf.RequestUpdatePause) * time.Millisecond)
	go app.callbackRequestHandler.ProcessRequests(time.Duration(conf.RequestCallBackUpdatePause) * time.Millisecond)

	app.botAPI, err = tgbotapi.NewBotAPI(conf.Token)
	if err != nil {
		return nil, fmt.Errorf("не удается инициализировать бота telegram: %v", err)
	}

	go app.HandleUpdates()
	go app.StartPeriodStatSending()

	go app.StartPeriodBirthSending()
	return &app, nil
}

// HandleUpdates запускает обработку всех обновлений поступающих боту из телеграмма
func (app *Bot) HandleUpdates() {
	// Настройка обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := app.botAPI.GetUpdatesChan(u)
	for update := range updates {

		go func(update tgbotapi.Update) {

			if update.SentFrom() != nil {
				user := model.User{
					UserName: update.SentFrom().UserName,
					UserID:   update.SentFrom().ID,
				}
				if user.UserID != 0 {
					// Проверяем существование только для ненулевых ID
					db.DB.Where("user_id = ?", user.UserID).FirstOrCreate(&user)
				} else {
					// Для нулевых ID просто создаем новую запись
					db.DB.Create(&user)
				}
			}

			// Обработка локальных стейтов
			if update.SentFrom() == nil {
				return
			}
			userStateName, err := app.GetUserState(update.SentFrom().ID)
			if err != nil {
				logger.Error("Ошибка при получении состояния пользователя:", err)
			}

			userState, ok := app.states[userStateName]
			if ok {
				app.HandleUserState(update, userState) // Обрабатываем состояние пользователя

			} else {
				fmt.Println("state не найден в мапе states:", userStateName) // Логируем, если состояние не найдено
			}

			// Обработка глобальных стейтов
			app.HandleGloblStates(update)
		}(update)

	}
}

func (app *Bot) GetUserState(userId int64) (string, error) {
	userStateInterface, ok := app.userStates.Get(strconv.FormatInt(userId, 10))
	if !ok {
		return "", fmt.Errorf("пользовательский статус не найден") // Обработка ошибки
	}

	userState, ok := userStateInterface.(string)
	if !ok {
		return "", fmt.Errorf("пользовательский статус не найден") // Обработка ошибки
	}

	return userState, nil // Возврат состояния пользователя
}

func (app *Bot) SetUserState(userId int64, state string) {
	key := strconv.FormatInt(userId, 10)
	app.userStates.Set(key, state, 24*time.Hour)
}

func (app *Bot) HandleUserState(update tgbotapi.Update, userState State) {
	fmt.Println("Выполнение действие при входе")
	// Если действие при входе разрешено
	if !userState.NotEntranceAction {
		fmt.Println("Выполнение действие при входе")
		if userState.AtEntranceFunc.Func != nil {
			userState.AtEntranceFunc.Func(app, update)
		} else {
			logger.Error("Попытка использования глобальной функции с :", userState)
		}

	}

	app.SelectHandler(update, userState)
}

func (app *Bot) SelectHandler(update tgbotapi.Update, userState State) {
	// fmt.Println("Поиск действия")
	switch {
	case update.Message != nil:
		app.handleMessage(userState, update)
	case update.CallbackQuery != nil:
		app.handleCallback(userState, update)
	}
}

func (app *Bot) HandleGloblStates(update tgbotapi.Update) {
	for _, state := range app.states {

		if state.Global {
			// Если действие при входе разрешено
			if !state.NotEntranceAction {
				state.AtEntranceFunc.Func(app, update)
			}
			// Если разрешено переходить в другие состояния
			if !state.NoContext {
				app.userStates.Set(strconv.FormatInt(update.SentFrom().ID, 10), state, 24*time.Hour)
			}
			app.SelectHandler(update, state)
		}
	}
}

// handleMessage ищет команду в map'е и выполняет ее
func (app *Bot) handleMessage(userState State, update tgbotapi.Update) {
	// Поиск события в map'е
	// fmt.Println("поиск события в map'e: " + strings.ToLower(strings.TrimSpace(update.Message.Text)))

	if currentAction, ok := userState.MessageRoute[strings.ToLower(strings.TrimSpace(update.Message.Text))]; ok {
		if err := currentAction.Func(app, update); err != nil {
			logger.Error("Ошибка при обработки команды ", update.Message.Text, " от пользователя (", update.Message.Chat.ID, ":", update.Message.Chat.UserName)
		} else {
			logger.Info("Успешно обработана команда: ", update.Message.Text, " от пользователя (", update.Message.Chat.ID, ":", update.Message.Chat.UserName)
		}
	} else {
		if userState.CatchAll {
			// logger.Info("Пользователь находился в событии, перехват CatcAll")
			userState.CatchAllFunc.Func(app, update)
		} else {
			logger.Info("Пользователь ( ", update.Message.Chat.ID, " : ", update.Message.Chat.UserName, " отправил команду ", update.Message.Text, ": в чат, которая не была найдена.")
		}

	}
}

// handleCallback ищет команду в map'е и выполняет ее
func (app *Bot) handleCallback(userState State, update tgbotapi.Update) {
	if update.CallbackQuery == nil {
		return
	}

	// Поиск обработчика в map'е callback обработчиков
	if currentAction, ok := userState.CallBackRoute[update.CallbackQuery.Data]; ok {
		// Обработчик найден
		if err := currentAction(app, update); err != nil {
			logger.Error("Ошибка при обработки Callback команды от пользователя (", update.CallbackQuery.From.ID, ":", update.CallbackQuery.From.UserName, err)
		} else {
			logger.Info("Успешно обработана Callback команда: ", update.CallbackQuery.Data, " от пользователя (", update.CallbackQuery.From.ID, ":", update.CallbackQuery.From.UserName)
		}
	} else {

		if userState.CatchAllCallBack {
			// logger.Info("Пользователь находился в событии, перехват CatchAllCallBack")
			userState.CatchAllCallBackfunc.Func(app, update)
		} else {
			// app.ShowAlert(update.CallbackQuery.ID, "Неизвестная команда")
		}

		logger.Info("Вызван callback метод: ", update.CallbackQuery.Data, "-  для которого не установлен обработчик. От пользователя (", update.CallbackQuery.From.ID, ":", update.CallbackQuery.From.UserName)
	}
}

// Отправляет статистику поданных заявок за текущие сутки
func (app *Bot) StartPeriodStatSending() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 21, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		duration := next.Sub(now)
		time.Sleep(duration)

		now = time.Now()
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())

		forms, transfer, urgentForms, err := db.GetFormsByDateRange(startDate, endDate)
		if err != nil {
			logger.Error("Не удалось получить статистику за определенный диапазон: ", err)
			continue
		}

		textMsg := fmt.Sprint("Статистика за ", startDate.Format("2006-01-02")) + fmt.Sprintf("\n\nКоличество замен: %d\n", forms)
		textMsg += fmt.Sprintf("Количество срочных замен: %d\n", transfer)
		textMsg += fmt.Sprintf("Количество переносов: %d\n", urgentForms)

		newMsg := tgbotapi.NewMessage(config.File.TelegramConfig.BotTgChat, textMsg)
		newMsg.ParseMode = "html"
		newMsg.ReplyToMessageID = config.File.TelegramConfig.StatTopicId

		_, err = app.SendMessage(newMsg)
		if err != nil {
			utils.HandleError(err)
		}
	}
}

// Отправляет информацию о ДР 2 раза в день. В 7 утра и 12 дня
func (app *Bot) StartPeriodBirthSending() {
	go func() {
		for {
			now := time.Now()
			nextMorning := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
			if now.After(nextMorning) {
				nextMorning = nextMorning.Add(24 * time.Hour)
			}
			duration := nextMorning.Sub(now)
			logger.Info("Sleeping until next morning:", nextMorning)
			time.Sleep(duration)

			logger.Info("Handling birthdays in the morning")
			HandleBirthDaysToTopic(app)
		}
	}()

	go func() {
		for {
			now := time.Now()
			nextNoon := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
			if now.After(nextNoon) {
				nextNoon = nextNoon.Add(24 * time.Hour)
			}
			duration := nextNoon.Sub(now)
			logger.Info("Sleeping until next noon:", nextNoon)
			time.Sleep(duration)

			logger.Info("Handling birthdays at noon")
			HandleBirthDaysToTopic(app)
		}
	}()
}
