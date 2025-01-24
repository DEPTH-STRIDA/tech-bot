package easycodeapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Member представляет информацию об участнике группы
type Member struct {
	StatusID         string `json:"status_id"`          // Идентификатор статуса
	OrderID          int    `json:"order_id"`           // Номер заказа
	Name             string `json:"name"`               // Имя родителя
	ChildName        string `json:"child_name"`         // Имя ребенка
	ChildAge         string `json:"child_age"`          // Возраст ребенка
	CountPayedLesson string `json:"count_payed_lesson"` // Количество оплаченных уроков
	Active           bool   `json:"active"`             // Активен ли участник
}

// GroupMembers представляет информацию об участниках группы
type GroupMembers struct {
	All       []Member            // Все участники группы
	Active    map[string][]Member // Активные участники, сгруппированные по статусам
	NotActive map[string][]Member // Неактивные участники, сгруппированные по статусам
}

// apiResponse представляет структуру ответа API
type apiResponse struct {
	Status  string `json:"status"`
	Members struct {
		All       json.RawMessage `json:"all"`
		Active    json.RawMessage `json:"active"`
		NotActive json.RawMessage `json:"not_active"`
	} `json:"members"`
}

// GetGroupMembers получает информацию об участниках конкретной группы
func (app *ApiClient) GetGroupMembers(groupNumber uint64) (*GroupMembers, error) {
	var result *GroupMembers

	err := app.request.HandleSyncRequest(func() error {
		members, err := app.fetchGroupMembers(groupNumber)
		if err != nil {
			return err
		}
		result = members
		app.logf("Получена информация об участниках группы %d", groupNumber)
		return nil
	})

	return result, err
}

// GetGroupsStats возвращает статистику по активным участникам в группах
// GetGroupsStats возвращает статистику по активным участникам в группах
func (app *ApiClient) GetGroupsStats(groupNumbers []uint64) (map[uint64]int, int, error) {
	if len(groupNumbers) == 0 {
		return nil, 0, fmt.Errorf("пустой список групп")
	}

	log.Println("Начало получения статистики по активным участникам в группах")
	totalActive := 0
	groupStats := make(map[uint64]int)

	for _, groupNum := range groupNumbers {
		log.Printf("Получение информации о группе %d", groupNum)
		members, err := app.GetGroupMembers(groupNum)
		if err != nil {
			log.Printf("Ошибка получения информации о группе %d: %v", groupNum, err)
			continue
		}

		activeCount := 0
		for _, member := range members.All {
			if member.Active {
				activeCount++
				totalActive++
			}
		}
		groupStats[groupNum] = activeCount
		log.Printf("Группа %d: %d активных участников", groupNum, activeCount)
	}

	log.Printf("Общее количество активных участников: %d", totalActive)
	return groupStats, totalActive, nil
}

// fetchGroupMembers выполняет запрос к API и обрабатывает полученные данные
func (app *ApiClient) fetchGroupMembers(groupNumber uint64) (*GroupMembers, error) {
	// Подготовка данных для запроса
	requestData := struct {
		CourseID    uint64 `json:"course_id"`
		AccessToken string `json:"access_token"`
	}{
		CourseID:    groupNumber,
		AccessToken: app.config.AccessToken,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("ошибка подготовки данных запроса: %v", err)
	}

	// Создание и выполнение запроса
	req, err := http.NewRequest("POST", app.config.MemberAPIURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}
	app.logf("Полученное тело ответа: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("некорректный код ответа: %d", resp.StatusCode)
	}

	// Разбор ответа
	var response apiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка разбора ответа: %v", err)
	}

	return parseGroupMembers(response.Members)
}

// parseGroupMembers разбирает данные об участниках из ответа API
func parseGroupMembers(data struct {
	All       json.RawMessage `json:"all"`
	Active    json.RawMessage `json:"active"`
	NotActive json.RawMessage `json:"not_active"`
}) (*GroupMembers, error) {
	result := &GroupMembers{
		Active:    make(map[string][]Member),
		NotActive: make(map[string][]Member),
	}

	// Разбор всех участников
	if len(data.All) > 0 {
		if err := json.Unmarshal(data.All, &result.All); err != nil {
			// Если не получилось как массив Member, пробуем как пустой массив
			var emptyArray []interface{}
			if err := json.Unmarshal(data.All, &emptyArray); err != nil {
				return nil, fmt.Errorf("ошибка разбора списка всех участников: %v", err)
			}
			// Если это пустой массив, оставляем пустой slice
			result.All = []Member{}
		}
	}

	// Разбор активных участников
	if len(data.Active) > 0 {
		// Сначала пробуем разобрать как map
		err := json.Unmarshal(data.Active, &result.Active)
		if err != nil {
			// Если не получилось как map, пробуем как массив
			var emptyArray []interface{}
			if err := json.Unmarshal(data.Active, &emptyArray); err != nil {
				return nil, fmt.Errorf("ошибка разбора списка активных участников: %v", err)
			}
			// Если это пустой массив, оставляем пустую map
		}
	}

	// Разбор неактивных участников
	if len(data.NotActive) > 0 {
		// Сначала пробуем разобрать как map
		err := json.Unmarshal(data.NotActive, &result.NotActive)
		if err != nil {
			// Если не получилось как map, пробуем как массив
			var emptyArray []interface{}
			if err := json.Unmarshal(data.NotActive, &emptyArray); err != nil {
				return nil, fmt.Errorf("ошибка разбора списка неактивных участников: %v", err)
			}
			// Если это пустой массив, оставляем пустую map
		}
	}

	return result, nil
}
