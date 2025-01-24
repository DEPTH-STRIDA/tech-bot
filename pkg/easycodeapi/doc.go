// Пакет easycodeapi предоставляет функциональность для взаимодействия с API школы EasyCode.
//
// Основные компоненты:
//
// 1. ApiClient - основной клиент для работы с API:
//   - Управляет запросами к API с помощью RequestHandler
//   - Обеспечивает логирование операций
//   - Поддерживает отложенную обработку запросов
//
// 2. Методы для работы с уроками:
//   - Получение информации об уроках
//   - Управление статусами уроков
//   - Работа с комментариями к урокам
//
// 3. Методы для работы с участниками:
//   - Получение информации об участниках
//   - Управление статусами участников
//   - Работа с данными участников
//
// Пример использования:
//
//	// Инициализация клиента
//	logger := myLogger // реализует interfaces.SimpleLogger
//	client, err := easycodeapi.NewEasyCodeApi(logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Получение информации об уроке
//	lesson, err := client.GetLesson(lessonID)
//	if err != nil {
//	    log.Printf("Ошибка получения урока: %v", err)
//	    return
//	}
//
//	// Работа с участником
//	member, err := client.GetMember(memberID)
//	if err != nil {
//	    log.Printf("Ошибка получения участника: %v", err)
//	    return
//	}
package easycodeapi
