package easycodeapi

import (
	"easycodeapp/internal/infrastructure/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// Lesson представляет структуру урока
type Lesson struct {
	Name              string `json:"name"`                 // Название курса
	LessonID          int64  `json:"id"`                   // ID урока
	CourseID          int64  `json:"course_id"`            // Номер курса (группы)
	DateStart         string `json:"date_start"`           // Дата-время урока
	TeacherID         int64  `json:"teacher_id"`           // ID учителя из CRM
	ActiveMemberCount int    `json:"active_members_count"` // Количество активных учеников
	LessonNumber      int64  `json:"lesson_number"`        // Номер урока
}

// GettAllLessons постепенными запросами вытаскивает все уроки по указанным преподавателям и обрабатывает их функцией
func (app *ApiClient) GettAllLessons(teachers []int64, date string, isNightRun bool, handler func([]Lesson, bool) error) []error {
	var errs []error

	limit := uint(50)

	offset := uint(0)
	// Шаг offset = 100
	for {
		// Выполняем запрос к API и получаем список уроков за определенный шаг offset
		currentLessons, err := app.getLesson(offset, limit, teachers, date)
		// Если произошла ошибка, то складируем ошибку
		if err != nil {
			errs = append(errs, err)
		}
		// Обрабатываем поступившие заявки
		err = handler(currentLessons, isNightRun)
		if err != nil {
			logger.Info("Ошибка при обработке списка уроков: ", err)
		}
		logger.Info("Собрано уроков: ", len(currentLessons))
		// logger.Info("Уроки: ", currentLessons)

		// Ломаем цикл, если запросы к API больше не приносят результатов
		if (len(currentLessons) == 0) || (currentLessons == nil) {
			break
		}
		// if len(currentLessons) < int(limit) {
		// 	break
		// }

		offset += limit
	}
	logger.Info("НАКОПИЛОСЬ ОШИБОК: ", errs)
	return nil
}

// getLesson синхронная функция для получения списка уроков по переданным фильтрам
func (app *ApiClient) getLesson(offset, limit uint, teachers []int64, date string) ([]Lesson, error) {
	var lessons []Lesson

	err := app.request.HandleSyncLowPriorityRequest(func() error {
		var err error
		lessons, err = app.getLessonRequest(offset, limit, teachers, date, app.config.AccessToken)
		return err
	})

	return lessons, err
}

// getLessonRequest выполняет запрос к API с переданным фильтром и возвращает список уроков
func (app *ApiClient) getLessonRequest(offset, limit uint, teachers []int64, date, accessToken string) ([]Lesson, error) {
	jsonData, err := getFilter(offset, limit, teachers, date, accessToken)

	if err != nil {
		return nil, err
	}
	// Создание запроса
	url := "https://school.easy-mo.ru/external/lessons/getAll"
	req, err := http.NewRequest("POST", url, strings.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Установка таймаута 30 секунд для клиента
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Info("Ошибка выполнения запроса:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Info("Ошибка чтения ответа:", err)
		return nil, err
	}

	// Логгирование тела ответа
	// logger.Info("Полученное тело ответа:", string(body))

	// Проверка кода ответа
	if resp.StatusCode == http.StatusForbidden {
		return nil, err
	}
	// logger.Error("ТЕЛО ТЕЛО ТЕЛ: ", string(body))

	Lessons, err := findAndParseJSON(string(body))
	if err != nil {
		return nil, err
	}
	return Lessons, nil
}

// getFilter возвращает настраенный фильтр для API запроса на получение списка уроков.
// Возвращает все уроки для указанных преподавателей, без дубликатов в указанную дату.
// Пример date: 2024-07-28T00:00:00.000Z
func getFilter(offset, limit uint, teachers []int64, date, accessToken string) (string, error) {
	teachersStr := getTeacherFilter(teachers...)
	jsonStr := fmt.Sprintf(`
	{
    "offset": %d,
    "filter": {
        "lesson": {
            "teachers": %s,
            "upcoming": false,
            "date_start": {
                "symbol": "=",
                "date": "%s"
            },
            "date_end": {
                "symbol": "=",
                "date": null
            },
            "date_start_ending": {
                "symbol": "=",
                "date": null
            },
            "date_end_ending": {
                "symbol": "=",
                "date": null
            },
            "lesson_number": null,
            "lessons_type": [],
            "exclude": true,
            "ignoreLessonVisit": null,
            "trackLessonVisits": false,
            "replace_teacher": null,
            "linkToLesson": null
        }
    },
    "limit": %d,
    "access_token": "%s"
}`, offset, teachersStr, date, limit, accessToken)

	return jsonStr, nil
}

// getTeacherFilter возвращает JSON строку формата [{"id":0000},{"id":00001}]
func getTeacherFilter(teachers ...int64) string {
	if len(teachers) == 0 {
		return "[]"
	}
	str := "["
	for i := range teachers {
		str += `{ "id":` + fmt.Sprint(teachers[i]) + "},"
	}
	str = str[:len(str)-1]
	str += "]"
	return str
}

// findAndParseJSON ищет JSON строку и парсит ее в слайс структур Lesson
func findAndParseJSON(text string) ([]Lesson, error) {
	var result []Lesson

	// Используем gjson для поиска JSON объекта "lessons"
	lessonsJSON := gjson.Get(text, "lessons")
	if !lessonsJSON.Exists() {
		return nil, fmt.Errorf("поле 'lessons' не найдено в тексте")
	}

	// Преобразуем найденный JSON в слайс структур Lesson
	if err := json.Unmarshal([]byte(lessonsJSON.Raw), &result); err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
	}

	// Исключаем элементы с active_members_count равным 0
	var filteredResult []Lesson
	for _, lesson := range result {
		if lesson.ActiveMemberCount != 0 {
			filteredResult = append(filteredResult, lesson)
		}
	}

	return filteredResult, nil
}
