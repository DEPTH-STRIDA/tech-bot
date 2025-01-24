package request

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"easycodeapp/pkg/logger/interfaces"
)

// RequestHandler управляет обработкой запросов с поддержкой приоритизации
// и ограничения скорости. Предоставляет два канала для запросов: обычный и низкоприоритетный.
type RequestHandler struct {
	requests            chan Request
	lowPriorityRequests chan Request
	ctx                 context.Context
	cancel              context.CancelFunc
	mu                  sync.Mutex
	isProcessing        bool
	logger              interface{} // может быть interfaces.BasicLogger, interfaces.SimpleLogger или nil
	loggingEnabled      bool
}

// NewRequestHandler создает новый экземпляр RequestHandler с заданной конфигурацией.
// Инициализирует каналы запросов и настраивает логирование согласно конфигурации.
// Возвращает ошибку в случае неудачной инициализации.
func NewRequestHandler(config Config) (*RequestHandler, error) {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &RequestHandler{
		requests:            make(chan Request, config.BufferSize),
		lowPriorityRequests: make(chan Request, config.BufferSize),
		ctx:                 ctx,
		cancel:              cancel,
		loggingEnabled:      true,
	}

	// Настройка логгера
	if v, ok := config.Logger.(bool); ok && !v {
		// Если Logger = false, отключаем логирование
		handler.loggingEnabled = false
	} else if config.Logger == nil {
		// Если Logger = nil, используем стандартный логгер
		handler.logger = log.New(os.Stdout, "request: ", log.LstdFlags)
	} else if l, ok := config.Logger.(interfaces.BasicLogger); ok {
		// Если передан BasicLogger
		handler.logger = l
	} else if l, ok := config.Logger.(interfaces.SimpleLogger); ok {
		// Если передан SimpleLogger
		handler.logger = l
	} else {
		return nil, errors.New("неподдерживаемый тип логгера")
	}

	return handler, nil
}

func (app *RequestHandler) log(msg string) {
	if !app.loggingEnabled {
		return
	}

	switch l := app.logger.(type) {
	case interfaces.SimpleLogger:
		l.Info(msg)
	case interfaces.BasicLogger:
		l.Print(msg)
	}
}

func (app *RequestHandler) logf(format string, args ...interface{}) {
	if !app.loggingEnabled {
		return
	}

	switch l := app.logger.(type) {
	case interfaces.SimpleLogger:
		l.Infof(format, args...)
	case interfaces.BasicLogger:
		l.Printf(format, args...)
	}
}

func (app *RequestHandler) logError(format string, args ...interface{}) {
	if !app.loggingEnabled {
		return
	}

	switch l := app.logger.(type) {
	case interfaces.SimpleLogger:
		l.Errorf(format, args...)
	case interfaces.BasicLogger:
		l.Printf("ERROR: "+format, args...)
	}
}

// HandleRequest добавляет запрос в канал обычного приоритета.
// Возвращает ошибку, если обработка не запущена.
func (app *RequestHandler) HandleRequest(req Request) error {
	app.mu.Lock()
	if !app.isProcessing {
		return errors.New("невозможно добавить запрос: обработка не запущена")
	}
	app.mu.Unlock()

	app.requests <- req
	return nil
}

// HandleLowPriorityRequest добавляет запрос в канал низкого приоритета.
// Возвращает ошибку, если обработка не запущена.
func (app *RequestHandler) HandleLowPriorityRequest(req Request) error {
	app.mu.Lock()
	if !app.isProcessing {
		return errors.New("невозможно добавить запрос: обработка не запущена")
	}
	app.mu.Unlock()

	app.lowPriorityRequests <- req
	return nil
}

// ProcessRequests запускает обработку запросов из обоих каналов с фиксированной паузой между запросами.
// Сначала обрабатываются запросы обычного приоритета, затем низкоприоритетные.
// Обработка продолжается до вызова StopProcessing или отмены контекста.
func (app *RequestHandler) ProcessRequests(pause time.Duration) {
	app.mu.Lock()
	if app.isProcessing {
		app.logf("Невозможно запустить обработку запросов: уже запущена")
		app.mu.Unlock()
		return
	}
	app.isProcessing = true
	app.mu.Unlock()

	for {
		select {
		case <-app.ctx.Done():
			app.isProcessing = false
			return
		case req := <-app.requests:
			err := req()
			if err != nil {
				app.logError("Ошибка выполнения запроса: %v", err)
			}
		case req := <-app.lowPriorityRequests:
			err := req()
			if err != nil {
				app.logError("Ошибка выполнения низкоприоритетного запроса: %v", err)
			}
		}
		time.Sleep(pause)
	}
}

// ProcessRequestsWithDynamicPause запускает обработку запросов с динамической регулировкой паузы.
// Длительность паузы увеличивается при последовательной обработке нескольких запросов,
// что помогает предотвратить перегрузку системы.
func (app *RequestHandler) ProcessRequestsWithDynamicPause(defaultPause time.Duration, incrementPause func(currentPause time.Duration) time.Duration) {
	app.mu.Lock()
	if app.isProcessing {
		app.logf("Невозможно запустить обработку запросов: уже запущена")
		app.mu.Unlock()
		return
	}
	app.isProcessing = true
	app.mu.Unlock()

	currentPause := defaultPause
	consecutiveRequests := 0

	for {
		select {
		case <-app.ctx.Done():
			app.isProcessing = false
			return
		case req := <-app.requests:
			consecutiveRequests++
			err := req()
			if err != nil {
				app.logError("Ошибка выполнения запроса: %v", err)
			}
		case req := <-app.lowPriorityRequests:
			consecutiveRequests++
			err := req()
			if err != nil {
				app.logError("Ошибка выполнения низкоприоритетного запроса: %v", err)
			}
		default:
			consecutiveRequests = 0
			currentPause = defaultPause
			time.Sleep(defaultPause)
			continue
		}

		if consecutiveRequests > 1 {
			currentPause = incrementPause(currentPause)
		} else {
			currentPause = defaultPause
		}

		time.Sleep(currentPause)
	}
}

// StopProcessing останавливает обработку запросов и освобождает ресурсы.
// После вызова этого метода новые запросы не будут обрабатываться до тех пор,
// пока не будет вызван ProcessRequests или ProcessRequestsWithDynamicPause.
func (app *RequestHandler) StopProcessing() {
	app.cancel()
	app.mu.Lock()
	app.isProcessing = false
	app.mu.Unlock()
}

// HandleSyncRequest отправляет запрос в очередь и ждет его выполнения.
// Возвращает ошибку, если запрос не удалось выполнить.
func (h *RequestHandler) HandleSyncRequest(fn func() error) error {
	var wg sync.WaitGroup
	var resultErr error

	wg.Add(1)
	h.HandleRequest(func() error {
		defer wg.Done()
		if err := fn(); err != nil {
			resultErr = err
			return err
		}
		return nil
	})

	wg.Wait()
	return resultErr
}

// HandleSyncLowPriorityRequest отправляет низкоприоритетный запрос в очередь и ждет его выполнения.
// Возвращает ошибку, если запрос не удалось выполнить.
func (h *RequestHandler) HandleSyncLowPriorityRequest(fn func() error) error {
	var wg sync.WaitGroup
	var resultErr error

	wg.Add(1)
	h.HandleLowPriorityRequest(func() error {
		defer wg.Done()
		if err := fn(); err != nil {
			resultErr = err
			return err
		}
		return nil
	})

	wg.Wait()
	return resultErr
}
