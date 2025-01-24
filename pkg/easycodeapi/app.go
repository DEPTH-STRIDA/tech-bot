package easycodeapi

import (
	"easycodeapp/pkg/logger/interfaces"
	"easycodeapp/pkg/request"
	"errors"
	"log"
	"os"
	"time"
)

// Config содержит настройки для клиента API
type Config struct {
	// Настройки API
	AccessToken     string // Токен доступа к API
	MemberAPIURL    string // URL для API работы с участниками
	LessonsAPIURL   string // URL для API работы с уроками
	ApiRequestPause int64  // Пауза между запросами в секундах
	ApiBufferSize   int    // Размер буфера запросов

	// Logger определяет способ логирования:
	// - nil: будет использован стандартный log.Logger
	// - false: логирование будет отключено
	// - interfaces.BasicLogger: будет использован базовый логгер
	// - interfaces.SimpleLogger: будет использован расширенный логгер
	Logger interface{}
}

// ApiClient структура для взаимодействия с API школы.
// Использует RequestHandler для отложенной обработки запросов
// и SimpleLogger для логирования операций.
type ApiClient struct {
	config         Config                  // Конфигурация для API
	request        *request.RequestHandler // Обработчик запросов
	logger         interface{}             // Логгер для записи информации о работе с API
	loggingEnabled bool                    // Флаг включенного логирования
}

// NewEasyCodeApi создает и возвращает новый экземпляр ApiClient.
// Принимает конфигурацию для настройки клиента.
// Логирование настраивается согласно переданному в конфигурации логгеру.
func NewEasyCodeApi(conf Config) (*ApiClient, error) {
	easyCodeApi := &ApiClient{
		config:         conf,
		loggingEnabled: true,
	}

	// Настройка логгера
	if v, ok := conf.Logger.(bool); ok && !v {
		// Если Logger = false, отключаем логирование
		easyCodeApi.loggingEnabled = false
	} else if conf.Logger == nil {
		// Если Logger = nil, используем стандартный логгер
		easyCodeApi.logger = log.New(os.Stdout, "EasyCodeAPI: ", log.LstdFlags)
	} else if l, ok := conf.Logger.(interfaces.BasicLogger); ok {
		// Если передан BasicLogger
		easyCodeApi.logger = l
	} else if l, ok := conf.Logger.(interfaces.SimpleLogger); ok {
		// Если передан SimpleLogger
		easyCodeApi.logger = l
	} else {
		return nil, errors.New("неподдерживаемый тип логгера")
	}

	// Настройка RequestHandler
	requestConfig := request.Config{
		BufferSize: conf.ApiBufferSize,
		Logger:     conf.Logger, // Передаем тот же логгер в RequestHandler
	}

	requestHandler, err := request.NewRequestHandler(requestConfig)
	if err != nil {
		return nil, err
	}
	easyCodeApi.request = requestHandler

	// Запуск процесса обработки запросов
	go easyCodeApi.request.ProcessRequests(time.Duration(conf.ApiRequestPause) * time.Second)

	if easyCodeApi.loggingEnabled {
		easyCodeApi.log("Запущена обработка api запросов")
	}
	return easyCodeApi, nil
}

// log логирует сообщение в зависимости от типа логгера
func (api *ApiClient) log(msg string) {
	if !api.loggingEnabled {
		return
	}

	switch l := api.logger.(type) {
	case interfaces.SimpleLogger:
		l.Info(msg)
	case interfaces.BasicLogger:
		l.Print(msg)
	case *log.Logger:
		l.Println(msg)
	}
}

// logf логирует форматированное сообщение в зависимости от типа логгера
func (api *ApiClient) logf(format string, args ...interface{}) {
	if !api.loggingEnabled {
		return
	}

	switch l := api.logger.(type) {
	case interfaces.SimpleLogger:
		l.Infof(format, args...)
	case interfaces.BasicLogger:
		l.Printf(format, args...)
	case *log.Logger:
		l.Printf(format, args...)
	}
}
