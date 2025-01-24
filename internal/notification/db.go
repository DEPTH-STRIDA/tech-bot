package notification

import (
	"easycodeapp/internal/config"
	"easycodeapp/internal/infrastructure/logger"
	m "easycodeapp/internal/model"
	"easycodeapp/pkg/model"
	"fmt"
	"reflect"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase() (*Database, error) {
	conf := config.File.DataBaseConfig
	dsn := ""
	if config.File.DataBaseConfig.Port == "" {
		dsn = fmt.Sprintf("host=%s user=%s dbname=%s password=%s sslmode=%s", conf.Host, conf.UserName, conf.DBName, conf.Password, conf.SSLMode)
	} else {
		dsn = fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", conf.Host, conf.Port, conf.UserName, conf.DBName, conf.Password, conf.SSLMode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	DB := &Database{
		DB: db,
	}

	return DB, nil
}

// CreateTable создает таблицу на основе переданной структуры
func (app *Database) CreateTable(model interface{}) error {
	return app.DB.AutoMigrate(model)
}

// DropTable удаляет таблицу, представленную моделью, переданной в качестве аргумента
func (app *Database) DropTable(model interface{}) error {
	// Удаляем таблицу
	if err := app.DB.Migrator().DropTable(model); err != nil {
		return err
	}
	return nil
}

// InsertRow вставляет строку в таблицу. Таблица определяет за счет тип структуры. Необходимо передать ссылку на структуру, которую надо вставить.
func (app *Database) InsertRow(value interface{}) error {
	return app.DB.Create(value).Error
}

// InsertRowUnique вставляет строку в таблицу только если нет строк с совпадающими значениями в указанных столбцах
func (app *Database) InsertRowUnique(value interface{}, uniqueFields map[string]string) error {

	// Создаем карту для хранения значений уникальных полей
	conditions := make(map[string]interface{})
	val := reflect.ValueOf(value).Elem()

	for structField, dbField := range uniqueFields {
		fieldValue := val.FieldByName(structField)
		if !fieldValue.IsValid() {
			return fmt.Errorf("поле %s не существует в структуре", structField)
		}
		if !fieldValue.CanInterface() {
			return fmt.Errorf("поле %s недоступно в структуре", structField)
		}
		conditions[dbField] = fieldValue.Interface()
	}

	// Проверяем существование строки с такими же значениями уникальных полей
	var count int64
	query := app.DB.Model(value)
	for dbField, val := range conditions {
		query = query.Where(fmt.Sprintf("%s = ?", dbField), val)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		// Строка с такими значениями уникальных полей уже существует
		return nil
	}

	// Вставляем новую строку
	return app.DB.Create(value).Error
}

// InsertRow вставляет строку в таблицу. Таблица определяет за счет тип структуры.Необходимо передать ссылку на структуру, которую надо вставить.
// Также надо указать поля, которые не должны совпдать с какой-либо строкой из таблицы

// GetRowByColumn получает запись из таблицы по значению указанного столбца
func (app *Database) GetRowByColumn(columnName string, value interface{}, row interface{}) error {
	// Выполняем запрос с использованием динамического имени столбца
	result := app.DB.Where(fmt.Sprintf("%s = ?", columnName), value).First(row)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (app *Database) DeleteRecordByColumn(columnName string, value interface{}, row interface{}) error {
	// Выполняем запрос на удаление записи из таблицы, указанной в модели
	result := app.DB.Where(fmt.Sprintf("%s = ?", columnName), value).Delete(row)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// UpdateRowConfig представляет конфигурацию для обновления записи
type UpdateRowConfig struct {
	SearchColumnName string      // Имя столбца для поиска
	SearchValue      interface{} // Значение для поиска
	NewColumnName    string      // Имя столбца для обновления
	NewValue         interface{} // Новое значение
	Row              interface{} // Структура, представляющая запись
}

// UpdateRow обновляет запись в таблице на основе переданного конфига
func (app *Database) UpdateRow(config UpdateRowConfig) error {

	// Создаем карту для обновления с использованием нового имени столбца
	updateValues := map[string]interface{}{
		config.NewColumnName: config.NewValue,
	}
	// Выполняем запрос для поиска и обновления записи
	result := app.DB.Model(config.Row).
		Where(fmt.Sprintf("%s = ?", config.SearchColumnName), config.SearchValue).
		UpdateColumns(updateValues)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (app *Database) HandleCallBackTable(columnName string, limit int, handler func(results []CallBack)) {
	offset := 0
	for {
		var results []CallBack

		err := app.DB.Transaction(func(tx *gorm.DB) error {
			err := app.getRecordsByColumn(tx, &CallBack{}, columnName, limit, offset, &results)
			if err != nil {
				return fmt.Errorf("не удалось получить партию: %w", err)
			}

			if len(results) > 0 {
				handler(results)
			}

			return nil
		})

		if err != nil {
			logger.Error("Ошибка при обработке партии CallBack: " + err.Error())
			return // Прерываем выполнение функции при ошибке
		}

		if len(results) == 0 || len(results) < limit {
			break // Выходим из цикла, если больше нет данных
		}

		offset += limit
	}
}

func (app *Database) HandleUserTable(columnName string, limit int, handler func(results []model.User)) {
	offset := 0
	for {
		var results []model.User

		err := app.DB.Transaction(func(tx *gorm.DB) error {
			err := app.getRecordsByColumn(tx, &model.User{}, columnName, limit, offset, &results)
			if err != nil {
				return fmt.Errorf("не удалось получить партию: %w", err)
			}

			if len(results) > 0 {
				handler(results)
			}

			return nil
		})

		if err != nil {
			logger.Error("Ошибка при обработке партии User: " + err.Error())
			return // Прерываем выполнение функции при ошибке
		}

		if len(results) == 0 || len(results) < limit {
			break // Выходим из цикла, если больше нет данных
		}

		offset += limit
	}
}

func (app *Database) HandleMessageTable(columnName string, limit int, handler func(results []m.Message)) {
	offset := 0
	for {
		var results []m.Message

		err := app.DB.Transaction(func(tx *gorm.DB) error {
			err := app.getRecordsByColumn(tx, &m.Message{}, columnName, limit, offset, &results)
			if err != nil {
				return fmt.Errorf("не удалось получить партию: %w", err)
			}

			if len(results) > 0 {
				handler(results)
			}

			return nil
		})

		if err != nil {
			logger.Error("Ошибка при обработке партии Message: " + err.Error())
			return // Прерываем выполнение функции при ошибке
		}

		if len(results) == 0 || len(results) < limit {
			break // Выходим из цикла, если больше нет данных
		}

		offset += limit
	}
}

func (app *Database) getRecordsByColumn(tx *gorm.DB, model interface{}, columnName string, limit, offset int, result interface{}) error {
	stmt := &gorm.Statement{DB: tx}
	_ = stmt.Parse(model)
	tableName := stmt.Schema.Table

	query := fmt.Sprintf(`SELECT * FROM "%s" ORDER BY "%s" LIMIT ? OFFSET ?`, tableName, columnName)
	dbResult := tx.Raw(query, limit, offset).Scan(result)
	if dbResult.Error != nil {
		return dbResult.Error
	}

	return nil
}

// GetRecordsByColumn возвращает срез по указанному столбцу
func (app *Database) GetRecordsByColumn(columnName string, value interface{}, result interface{}) error {

	// Проверяем, что result является указателем на срез нужной структуры
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result должен быть указателем на срез структур")
	}

	sliceType := resultValue.Elem().Type().Elem()
	if sliceType.Kind() != reflect.Struct {
		return fmt.Errorf("result должен быть срезом структур")
	}

	// Выполняем запрос к базе данных
	err := app.DB.Where(fmt.Sprintf("%s = ?", columnName), value).Find(result).Error
	if err != nil {
		return err
	}

	return nil
}

// GetColumnValues позволяет получить все значения из указанного столбца таблицы "users" и поместить их в срез, не зависимо от типа данных в этом столбце. Она использует рефлексию, чтобы работать с динамическими типами.
func (app *Database) GetColumnValues(columnName string, result interface{}) error {

	// Проверяем, что result является указателем на срез
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("result должен быть указателем на срез")
	}

	// Выполняем запрос к базе данных, чтобы получить все значения из столбца
	rows, err := app.DB.Table("users").Select(columnName).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	// Создаем новый срез нужного типа
	sliceType := resultValue.Elem().Type().Elem()
	resultSlice := reflect.MakeSlice(resultValue.Elem().Type(), 0, 0)

	// Заполняем срез полученными значениями
	for rows.Next() {
		// Создаем новый элемент нужного типа
		elem := reflect.New(sliceType).Elem()

		// Сканируем значение из базы данных в элемент
		if err := rows.Scan(elem.Addr().Interface()); err != nil {
			return err
		}

		// Добавляем элемент в срез
		resultSlice = reflect.Append(resultSlice, elem)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Присваиваем срез в result
	resultValue.Elem().Set(resultSlice)
	return nil
}
