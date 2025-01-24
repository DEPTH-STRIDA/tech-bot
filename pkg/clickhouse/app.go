/*
Предоставляет структуру, которая может отправить данные в клик хаус
*/
package clickhouse

import (
	"crypto/tls"
	"crypto/x509"
	"easycodeapp/pkg/logger/interfaces"
	"easycodeapp/pkg/model"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type ClickHouseServer interface {
	Connect() error
	Disconnect()
	InsertRequests([]model.Form) error
}

type Config struct {
	DBHost                   string
	DBName                   string
	DBUser                   string
	DBPort                   string
	DBPass                   string
	DBNumberRepetitions      int
	DBFormTableName          string
	CHPauseAfterSQLExecute   float32
	PauseAfterFailConnection int

	DB *gorm.DB
	// Logger определяет способ логирования:
	// - nil: будет использован стандартный log.Logger
	// - false: логирование будет отключено
	// - interfaces.BasicLogger: будет использован базовый логгер
	// - interfaces.SimpleLogger: будет использован расширенный логгер
	Logger interface{}
}

// ClickHouse представляет клиент для работы с ClickHouse
type ClickHouse struct {
	sync.Mutex
	config     Config
	db         *gorm.DB
	connection *http.Client

	// Logger определяет способ логирования:
	// - nil: будет использован стандартный log.Logger
	// - false: логирование будет отключено
	// - interfaces.BasicLogger: будет использован базовый логгер
	// - interfaces.SimpleLogger: будет использован расширенный логгер
	logger         interface{}
	loggingEnabled bool
}

// NewClickHouse создает новый экземпляр ClickHouse с заданной конфигурацией.
// Он настраивает логгер в зависимости от переданной конфигурации.
// Возвращает ошибку, если тип логгера не поддерживается.
func NewClickHouse(config Config) (*ClickHouse, error) {

	clickHouse := &ClickHouse{
		config:         config,
		db:             config.DB,
		loggingEnabled: true,
	}

	// Настройка логгера
	if v, ok := config.Logger.(bool); ok && !v {
		// Если Logger = false, отключаем логирование
		clickHouse.loggingEnabled = false
	} else if config.Logger == nil {
		// Если Logger = nil, используем стандартный логгер
		clickHouse.logger = log.New(os.Stdout, "request: ", log.LstdFlags)
	} else if l, ok := config.Logger.(interfaces.BasicLogger); ok {
		// Если передан BasicLogger
		clickHouse.logger = l
	} else if l, ok := config.Logger.(interfaces.SimpleLogger); ok {
		// Если передан SimpleLogger
		clickHouse.logger = l
	} else {
		return nil, errors.New("неподдерживаемый тип логгера")
	}

	clickHouse.log("Успешно создана структура для работы с ClickHouse")
	return clickHouse, nil
}

// log записывает сообщение в лог, если логирование включено.
// Использует разные методы в зависимости от типа логгера.
func (app *ClickHouse) log(msg string) {
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

// logf записывает форматированное сообщение в лог, если логирование включено.
// Использует разные методы в зависимости от типа логгера.
func (app *ClickHouse) logf(format string, args ...interface{}) {
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

func (app *ClickHouse) logError(format string, args ...interface{}) {
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

// getDefaultRequest создает новый HTTP запрос с заданными заголовками
func (app *ClickHouse) getDefaultRequest(text string) (*http.Request, error) {
	conf := app.config

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s:%s/", conf.DBHost, conf.DBPort), strings.NewReader(text))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-ClickHouse-User", conf.DBUser)
	req.Header.Add("X-ClickHouse-Key", conf.DBPass)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

// connectCert устанавливает SSL соединение с ClickHouse
func (app *ClickHouse) connectCert() error {
	caCert, err := os.ReadFile("cert/RootCA.crt")
	if err != nil {
		return err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	app.connection = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
	return nil
}

// Connect устанавливает соединение с ClickHouse
func (app *ClickHouse) Connect() error {
	var err error
	for i := 0; i < app.config.DBNumberRepetitions; i++ {
		err = app.connectCert()
		if err != nil {
			app.logError("Попытка подключения к БД клик хаус №", i+1, " НЕ УДАЛАСЬ: ", err)
			time.Sleep(time.Second * time.Duration(app.config.PauseAfterFailConnection))
		} else {
			return nil
		}
	}
	return err
}

// Disconnect закрывает соединение с ClickHouse
func (app *ClickHouse) Disconnect() {
	if app.connection != nil {
		app.connection.CloseIdleConnections()
		app.connection = nil
	}
}

// executeSQL выполняет запрос SQL и возвращает ответ
func (app *ClickHouse) executeSQL(sql string) (*http.Response, error) {
	if err := app.Connect(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к ClickHouse: %v", err)
	}
	defer app.Disconnect()

	req, err := app.getDefaultRequest(sql)
	if err != nil {
		return nil, err
	}
	app.Lock()
	resp, err := app.connection.Do(req)
	if err != nil {
		app.Unlock()
		return nil, err
	}
	time.Sleep(time.Duration(app.config.CHPauseAfterSQLExecute) * time.Second)
	app.Unlock()

	return resp, nil
}

// handleResponse обрабатывает ответ от ClickHouse
func (app *ClickHouse) handleResponse(resp *http.Response) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("ошибка при отправке данных в ClickHouse. Код ответа: %d. Тело ответа: %s", resp.StatusCode, string(data))
	}
	app.log("Успешно отправлены данные в ClickHouse. Тело ответа: " + string(data))
	return nil
}
