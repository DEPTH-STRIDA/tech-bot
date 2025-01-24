package clickhouse

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func (app *ClickHouse) GetLast500FormsFromClickHouse() error {
	conf := app.config

	if err := app.Connect(); err != nil {
		return fmt.Errorf("не удалось подключиться к ClickHouse: %v", err)
	}
	defer app.Disconnect()

	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` ORDER BY id DESC LIMIT 500 FORMAT TabSeparated", conf.DBName, conf.DBFormTableName)

	req, err := app.getDefaultRequest(query)
	if err != nil {
		return fmt.Errorf("ошибка при создании запроса: %v", err)
	}

	resp, err := app.connection.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка при получении данных из ClickHouse. Код ответа: %d. Тело ответа: %s", resp.StatusCode, string(body))
	}

	file, err := os.Create("last_500_forms_from_clickhouse.txt")
	if err != nil {
		return fmt.Errorf("ошибка при создании файла: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(resp.Body)
	headers := []string{}
	formCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		if len(headers) == 0 {
			headers = fields
			continue
		}

		formCount++
		fmt.Fprintf(file, "Форма #%d\n", formCount)
		for i, header := range headers {
			if i < len(fields) {
				fmt.Fprintf(file, "%s: %s\n", header, fields[i])
			}
		}
		fmt.Fprintln(file, "---")
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка при чтении ответа: %v", err)
	}

	app.log(fmt.Sprintf("Последние %d форм из ClickHouse успешно записаны в файл 'last_500_forms_from_clickhouse.txt' в %s", formCount, time.Now().Format("2006-01-02 15:04:05")))
	return nil
}

// Новый метод для удаления заявок
func (app *ClickHouse) DeleteProcessedForms(formIDs []uint) error {
	conf := app.config

	if len(formIDs) == 0 {
		return nil
	}

	// Формируем строку с ID для SQL запроса
	idStr := make([]string, len(formIDs))
	for i, id := range formIDs {
		idStr[i] = fmt.Sprintf("%d", id)
	}

	// SQL запрос для удаления
	deleteSQL := fmt.Sprintf("ALTER TABLE `%s`.`%s` DELETE WHERE id IN (%s)",
		conf.DBName, conf.DBFormTableName, strings.Join(idStr, ","))

	app.log("Подготовлен запрос на удаление: " + deleteSQL)

	resp, err := app.executeSQL(deleteSQL)
	if err != nil {
		return fmt.Errorf("ошибка выполнения SQL для удаления: %v", err)
	}
	defer resp.Body.Close()

	return app.handleResponse(resp)
}

func (app *ClickHouse) GetAllFormsWithID() error {
	conf := app.config

	if err := app.Connect(); err != nil {
		return fmt.Errorf("не удалось подключиться к ClickHouse: %v", err)
	}
	defer app.Disconnect()

	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` FORMAT TabSeparated", conf.DBName, conf.DBFormTableName)

	req, err := app.getDefaultRequest(query)
	if err != nil {
		return fmt.Errorf("ошибка при создании запроса: %v", err)
	}

	resp, err := app.connection.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка при получении данных из ClickHouse. Код ответа: %d. Тело ответа: %s", resp.StatusCode, string(body))
	}

	file, err := os.Create("all_forms_with_id_from_clickhouse.txt")
	if err != nil {
		return fmt.Errorf("ошибка при создании файла: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(resp.Body)
	headers := []string{}
	formCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		if len(headers) == 0 {
			headers = fields
			fmt.Fprintln(file, "Заголовки:")
			fmt.Fprintln(file, strings.Join(headers, ", "))
			fmt.Fprintln(file, "---")
			continue
		}

		formCount++
		fmt.Fprintf(file, "Форма #%d\n", formCount)
		for i, header := range headers {
			if i < len(fields) {
				fmt.Fprintf(file, "%s: %s\n", header, fields[i])
			}
		}
		fmt.Fprintln(file, "---")
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка при чтении ответа: %v", err)
	}

	app.log(fmt.Sprintf("Все формы (%d штук) из ClickHouse успешно записаны в файл 'all_forms_with_id_from_clickhouse.txt' в %s", formCount, time.Now().Format("2006-01-02 15:04:05")))
	return nil
}

// DeleteSpecificRecords удаляет конкретные записи из таблицы ClickHouse
func (app *ClickHouse) DeleteSpecificRecords() error {
	deleteSQL := `ALTER TABLE easycode-analytic.replace_transfer_forms
    DELETE WHERE form_date = '2024-09-24' AND form_time IN (
        '2024-09-24 23:18:05',
        '2024-09-24 23:07:08',
        '2024-09-24 23:19:42',
        '2024-09-24 23:09:15'
    )`

	app.log("Подготовлен запрос на удаление конкретных записей")
	app.log("SQL запрос: " + deleteSQL)

	// Устанавливаем соединение с ClickHouse
	if err := app.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к ClickHouse: %v", err)
	}
	defer app.Disconnect()

	// Выполняем SQL-запрос
	resp, err := app.executeSQL(deleteSQL)
	if err != nil {
		return fmt.Errorf("ошибка выполнения SQL для удаления конкретных записей: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем ответ
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка при удалении записей из ClickHouse. Код ответа: %d. Тело ответа: %s", resp.StatusCode, string(body))
	}

	app.log("Конкретные записи успешно удалены из ClickHouse")
	return nil
}

// DeleteAllRecords удаляет все записи из таблицы ClickHouse
func (app *ClickHouse) DeleteAllRecords() error {
	conf := app.config

	// SQL запрос для удаления всех записей
	deleteSQL := fmt.Sprintf("ALTER TABLE `%s`.`%s` DELETE WHERE 1=1",
		conf.DBName, conf.DBFormTableName)

	app.log("Подготовлен запрос на удаление всех записей из таблицы")
	app.log("SQL запрос: " + deleteSQL)

	// Запрашиваем подтверждение
	var confirm string
	fmt.Print("Вы уверены, что хотите удалить ВСЕ записи из таблицы? (y/n): ")
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		return fmt.Errorf("операция отменена пользователем")
	}

	// Устанавливаем соединение с ClickHouse
	if err := app.Connect(); err != nil {
		return fmt.Errorf("ошибка подключения к ClickHouse: %v", err)
	}
	defer app.Disconnect()

	// Выполняем SQL-запрос
	resp, err := app.executeSQL(deleteSQL)
	if err != nil {
		return fmt.Errorf("ошибка выполнения SQL для удаления всех записей: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем ответ
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка при удалении записей из ClickHouse. Код ответа: %d. Тело ответа: %s", resp.StatusCode, string(body))
	}

	app.log("Все записи успешно удалены из таблицы ClickHouse")
	return nil
}
