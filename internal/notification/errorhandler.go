package notification

import (
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/internal/tg"
	"fmt"
)

const (
	ErrorParsingEmptyCRMID = iota
	ErrorParsingCRMID
	ErrorInsertIntoTeacherDB
	ErrorPastChatIdInt64
)

func (m *NotificationManager) handleNotificationError(err error, value interface{}, errType int) error {
	if err == nil {
		return nil
	}

	nerr := &NotificationError{
		Type:  errType,
		Value: value,
		Err:   err,
	}

	logger.Error(nerr.Error())

	if errType != ErrorInsertIntoTeacherDB {
		tg.TelegramBot.SendAllAdmins(nerr.Error())
	}
	return nerr
}

type NotificationError struct {
	Type  int
	Value interface{}
	Err   error
}

func (e *NotificationError) Error() string {
	switch e.Type {
	case ErrorParsingEmptyCRMID:
		valueTyped, ok := e.Value.([]string)
		if !ok || len(valueTyped) < 2 {
			return fmt.Sprintf("Не удалось распарсить CRM ID: пустая строка\nИмя преподавателя: %v;\nTG ID чата: %v\nОшибка: %v\nПреподаватель не будет получать уведомления об уроках", valueTyped[0], valueTyped[1], e.Err)
		}
		return ""
	case ErrorParsingCRMID:
		valueTyped, ok := e.Value.([]string)
		if !ok || len(valueTyped) < 3 {
			return fmt.Sprintf("Не удалось распарсить CRM ID: %v\nИмя преподавателя: %v;\nTG ID чата: %v\nОшибка: %v\nПреподаватель не будет получать уведомления об уроках", valueTyped[0], valueTyped[1], valueTyped[2], e.Err)
		}
		return ""
	case ErrorInsertIntoTeacherDB:
		valueTyped, ok := e.Value.([]string)
		if !ok || len(valueTyped) < 3 {
			return fmt.Sprintf("Не удалось вставить преподавателя в базу данных преподавателей:\nПреподватель: %+v\nОшибка: %v", valueTyped, e.Err)
		}
		return fmt.Sprintf("Не удалось вставить преподавателя в базу данных преподавателей:\nПреподватель: %+v\nОшибка: %v", valueTyped, e.Err)
	case ErrorPastChatIdInt64:
		return fmt.Sprintf("Не удалось распарсить chat id преподавателя\nВозможно преподаватель не имеет ЛК\nПреподватель:<strong> %+v</strong>\nОшибка: <strong>%v</strong>", e.Value, e.Err)
	default:
		return fmt.Sprintf("Неизвестная ошибка: %v", e.Err)
	}
}
