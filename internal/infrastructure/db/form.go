package db

import (
	"easycodeapp/internal/infrastructure/logger"
	"easycodeapp/pkg/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

// CreateFormIfNoSimilar проверяет наличие похожей заявки у пользователя по TelegramID и создаёт новую форму только если копии нет.
// Это позволяет снизить нагрузку на базу данных, выполняя проверку и создание в одном шаге.
func CreateFormIfNoSimilar(newForm model.Form, duration time.Duration) (*model.Form, error) {
	var existingForm model.Form

	// Рассчитываем время отсечения
	cutoffTime := time.Now().Add(-duration)

	// Начинаем транзакцию
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Проверяем наличие похожей формы у пользователя за указанный период
		err := tx.Where("telegram_id = ? AND created_at >= ?", newForm.TelegramUserID, cutoffTime).
			Where("lesson_date = ? AND lesson_time = ? AND replace_format = ? AND group_number = ? AND teacher = ? AND subject = ? AND module = ? AND lesson = ? AND reason = ? AND replace_transfer = ? AND link = ? AND imp_info = ? AND mentoring_inf_1 = ? AND mentoring_inf_2 = ? AND mentoring_inf_3 = ? AND transfer_time = ? AND team_leader = ?",
				newForm.LessonDate,
				newForm.LessonTime,
				newForm.ReplaceFormat,
				newForm.GroupNumber,
				newForm.Teacher,
				newForm.Subject,
				newForm.Module,
				newForm.Lesson,
				newForm.Reason,
				newForm.ReplaceTransfer,
				newForm.Link,
				newForm.ImpInfo,
				newForm.MentoringInf1,
				newForm.MentoringInf2,
				newForm.MentoringInf3,
				newForm.TransferTime,
				newForm.TeamLeader).
			First(&existingForm).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err // Возвращаем ошибку транзакции
		}

		if err == nil {
			// Похожая форма уже существует
			return errors.New("похожая форма уже существует")
		}

		// Похожая форма не найдена, создаём новую форму
		if err := tx.Create(&newForm).Error; err != nil {
			return err // Возвращаем ошибку транзакции
		}

		return nil // Успешное выполнение транзакции
	})

	if err != nil {
		return nil, err
	}

	return &newForm, nil
}

// GetForm возвращает заявку из БД, если она не старше указанного периода
func GetForm(id int64, duration time.Duration) (*model.Form, error) {
	var form model.Form
	// Рассчитываем время отсечения
	cutoffTime := time.Now().Add(-duration)

	if err := DB.Where("id = ? AND created_at >= ?", id, cutoffTime).First(&form).Error; err != nil {
		return nil, err
	}
	return &form, nil
}

// DeleteFormByID удаляет форму из базы данных по заданному ID
func DeleteForm(id int64) error {
	logger.Info("Удаление формы с ID: ", id)
	result := DB.Delete(&model.Form{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetActiveUserForms получает заявки пользователя по TelegramID, не старее указанного Duration и не удаленные
func GetActiveUserForms(telegramID int64, duration time.Duration) ([]model.Form, error) {
	logger.Info("Начало выполнения GetActiveUserForms")
	logger.Debug("Параметры вызова - TelegramID: ", telegramID, ", Duration: ", duration)

	var forms []model.Form

	// Рассчитываем время отсечения
	cutoffTime := time.Now().Add(-duration)
	logger.Debug("Вычислен cutoffTime: ", cutoffTime.Format(time.RFC3339))

	// Выполняем запрос к базе данных
	logger.Info("Выполнение запроса к базе данных для получения активных форм пользователя")
	err := DB.Where("telegram_id = ? AND created_at >= ?", telegramID, cutoffTime).Find(&forms).Error

	if err != nil {
		logger.Error("Ошибка при выполнении запроса к базе данных: ", err)
		return nil, err
	}
	logger.Debug("Количество найденных форм: ", len(forms))

	// Проверяем, найдены ли какие-либо формы
	if len(forms) == 0 {
		logger.Warn("Нет активных форм для TelegramID: ", telegramID, " за период: ", duration)
		return nil, gorm.ErrRecordNotFound
	}

	// Логируем найденные формы (можно ограничить количество или скрыть чувствительные данные)
	for i, form := range forms {
		logger.Debug("Форма ", i+1, ": ", form)
	}

	logger.Info("Успешное завершение GetActiveUserForms")
	return forms, nil
}

// UpdateFormByTelegramID обновляет определённые поля формы по TelegramID и UID
func UpdateFormByTelegramID(telegramID int64, updatedFields map[string]interface{}) error {
	result := DB.Model(&model.Form{}).
		Where("telegram_id = ?", telegramID).
		Updates(updatedFields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateFormByTelegramID обновляет определённые поля формы по TelegramID и UID
func UpdateFormByID(ID int64, updatedFields map[string]interface{}) error {
	result := DB.Model(&model.Form{}).
		Where("id = ?", ID).
		Updates(updatedFields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// OverwriteFormByID полностью переписывает все поля формы по её ID
func OverwriteFormByID(formID int64, newForm model.Form) error {
	// Устанавливаем ID формы, чтобы GORM знал, какую запись обновлять
	newForm.ID = formID

	// Используем Save для обновления всех полей формы
	if err := DB.Save(&newForm).Error; err != nil {
		return err
	}
	return nil
}

// GetFormsForGoogleSheet возвращает формы, которые не удалены, GoogleSheetStatus соответствует заданному значению и CreatedAt старее указанного времени
func GetFormsForGoogleSheet(googleSheetStatus bool, cutoffTime time.Time) ([]model.Form, error) {
	logger.Info("Начало выполнения GetFormsForGoogleSheet")
	logger.Debug("Параметры вызова - GoogleSheetStatus: ", googleSheetStatus, ", CutoffTime: ", cutoffTime.Format(time.RFC3339))

	var forms []model.Form

	err := DB.Where("google_sheet_status = ? AND created_at <= ? AND deleted_at IS NULL", googleSheetStatus, cutoffTime).
		Find(&forms).Error

	if err != nil {
		logger.Error("Ошибка при выполнении запроса к базе данных в GetFormsForGoogleSheet: ", err)
		return nil, err
	}

	logger.Debug("Количество найденных форм: ", len(forms))
	logger.Info("Успешное завершение GetFormsForGoogleSheet")
	return forms, nil
}

// UpdateGoogleSheetStatus обновляет статус GoogleSheetStatus и номер строки GoogleSheetLineNumber для указанной формы
func UpdateGoogleSheetStatus(formID int64, status bool, lineNumber int) error {
	logger.Info("Начало выполнения UpdateGoogleSheetStatus для формы ID ", formID)
	logger.Debug("Новый статус GoogleSheetStatus: ", status, ", GoogleSheetLineNumber: ", lineNumber)

	// Проверка существования формы перед обновлением
	var form model.Form
	err := DB.Where("id = ?", formID).First(&form).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("Форма с ID ", formID, " не найдена.")
			return gorm.ErrRecordNotFound
		}
		logger.Error("Ошибка при поиске формы с ID ", formID, ": ", err)
		return err
	}
	logger.Debug("Форма найдена: ", form)

	// Обновление только указанной формы
	result := DB.Model(&model.Form{}).
		Where("id = ?", formID).
		Updates(map[string]interface{}{
			"google_sheet_status":      status,
			"google_sheet_line_number": lineNumber,
		})

	if result.Error != nil {
		logger.Error("Ошибка при обновлении GoogleSheetStatus и GoogleSheetLineNumber для формы ID ", formID, ": ", result.Error)
		return result.Error
	}

	// Логирование количества затронутых строк
	logger.Info("Количество обновленных строк: ", result.RowsAffected)

	if result.RowsAffected == 0 {
		logger.Warn("Не удалось обновить форму с ID ", formID, ". Проверьте условия обновления.")
		return errors.New("не удалось обновить форму")
	}

	logger.Info("GoogleSheetStatus для формы ID ", formID, " успешно обновлен на ", status,
		" и GoogleSheetLineNumber установлен на ", lineNumber)
	return nil
}

// GetFirstFormForGoogleSheet возвращает первую подходящую форму для отправки в Google Sheets
func GetFirstFormForGoogleSheet(googleSheetStatus bool, cutoffTime time.Time) (*model.Form, error) {
	logger.Info("Начало выполнения GetFirstFormForGoogleSheet")
	logger.Debug("Параметры вызова - GoogleSheetStatus: ", googleSheetStatus, ", CutoffTime: ", cutoffTime.Format(time.RFC3339))

	var form model.Form
	err := DB.Where("google_sheet_status = ? AND created_at <= ? AND deleted_at IS NULL", googleSheetStatus, cutoffTime).First(&form).Error
	if err != nil {
		return nil, err
	}
	logger.Debug("Найдена форма ID: ", form.ID)

	return &form, nil
}

// GetNextGoogleSheetLineNumber возвращает следующий уникальный номер строки для Google Sheets
func GetNextGoogleSheetLineNumber() (int, error) {
	var maxLineNumber int
	err := DB.Model(&model.Form{}).Select("MAX(google_sheet_line_number)").Scan(&maxLineNumber).Error
	if err != nil {
		logger.Error("Ошибка при получении максимального номера строки Google Sheets: ", err)
		return 0, err
	}
	return maxLineNumber + 1, nil
}

// GetFormsByDateRange возвращает слайс заявок за указанный диапазон дат
func GetFormsByDateRange(startDate time.Time, endDate time.Time) (int, int, int, error) {
	var transfer, urgentForms int
	var forms []model.Form

	err := DB.Model(&model.Form{}).
		Where("lesson_date >= ? AND lesson_date < ?", startDate, endDate).
		Find(&forms).Error
	if err != nil {
		return 0, 0, 0, err
	}

	for _, form := range forms {
		if form.IsEmergency {
			urgentForms++
		}
		if form.ReplaceTransfer == "ПЕРЕНОС" {
			transfer++
		}
	}

	return len(forms), transfer, urgentForms, nil
}
