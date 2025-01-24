// Пакет interfaces определяет стандартные интерфейсы для логирования,
// которые могут использоваться различными компонентами приложения.
//
// Пакет предоставляет несколько уровней абстракции для логирования:
//
// 1. BasicLogger - базовый интерфейс, совместимый со стандартным log.Logger
//
// 2. LevelLogger и FormattedLevelLogger - интерфейсы для логирования с уровнями
// (Info, Error, Debug, Warn, Fatal) и их форматированные версии
//
// 3. StackTraceLogger - интерфейс для логирования со стектрейсом
//
// 4. ContextLogger - интерфейс для логирования с дополнительными полями контекста
//
// Также предоставляются комбинированные интерфейсы:
//   - Logger - полный интерфейс, включающий все возможности
//   - SimpleLogger - базовый набор возможностей без стектрейса и контекста
//
// Использование:
//
//	type MyService struct {
//	    log interfaces.SimpleLogger
//	}
//
//	func NewMyService(logger interfaces.SimpleLogger) *MyService {
//	    return &MyService{log: logger}
//	}
package interfaces
