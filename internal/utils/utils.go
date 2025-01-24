package utils

import (
	"easycodeapp/internal/infrastructure/logger"
	"fmt"
	"time"
)

// HandlerError логгирует ошибку в случае ее наличия и возвращает ее
func HandleError(err error) error {
	if err != nil {
		logger.Error(err)
		return err
	}
	return nil
}

// setLocationTime устанавливает часовой пояс по умолчанию для глобальной переменной.
// Принимает строку с названием локации и возвращает ошибку, если часовой пояс не удалось загрузить.
func InitGlobalLocationTime() error {
	// Устанавливаем локацию по умолчанию для time.Local
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return fmt.Errorf("ошибка при смене локации на %s: %w", "Europe/Moscow", err)
	}
	time.Local = loc
	return nil
}

// CalculateRemainingTime возвращает строку с оставшимся временем для редактирования заявки.
// remainingDuration — продолжительность времени жизни кеша.
func CalculateRemainingTime(createdAt time.Time, remainingDuration time.Duration) string {
	expirationTime := createdAt.Add(remainingDuration)
	now := time.Now()

	if now.After(expirationTime) {
		return "00:00" // Время истекло
	}

	duration := expirationTime.Sub(now)

	totalSeconds := int(duration.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Функция для вычисления разницы между двумя датами
func CalculateDifference(start, end time.Time) (int, int, int) {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	days := end.Day() - start.Day()

	if days < 0 {
		months--
		// Получаем последний день предыдущего месяца
		lastMonth := end.AddDate(0, -1, 0)
		days += time.Date(lastMonth.Year(), lastMonth.Month()+1, 0, 0, 0, 0, 0, lastMonth.Location()).Day()
	}

	if months < 0 {
		years--
		months += 12
	}

	return years, months, days
}

// parseDate пытается распарсить дату в нескольких форматах
func ParseDate(dateStr string) (time.Time, error) {
	formats := []string{"02.01.2006", "2.1.2006", "02.01.06", "2.1.06"}
	var parsedDate time.Time
	var err error

	for _, format := range formats {
		parsedDate, err = time.Parse(format, dateStr)
		if err == nil {
			return parsedDate, nil
		}
	}

	return time.Time{}, fmt.Errorf("не удалось распарсить дату: %s", dateStr)
}
