package request

// Config определяет параметры конфигурации для RequestHandler.
type Config struct {
	// BufferSize определяет размер каналов для запросов.
	BufferSize int

	// Logger определяет способ логирования:
	// - nil: будет использован стандартный log.Logger
	// - false: логирование будет отключено
	// - interfaces.BasicLogger: будет использован базовый логгер
	// - interfaces.SimpleLogger: будет использован расширенный логгер
	Logger interface{}
}

// DefaultConfig возвращает новый экземпляр Config со стандартными настройками.
// По умолчанию используется стандартный log.Logger с размером буфера в 1000 запросов.
func DefaultConfig() Config {
	return Config{
		BufferSize: 1000,
		Logger:     nil, // будет использован стандартный логгер
	}
}

// DisableLogging возвращает конфигурацию с отключенным логированием.
func DisableLogging() Config {
	return Config{
		BufferSize: 1000,
		Logger:     false,
	}
}
