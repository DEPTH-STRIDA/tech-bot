package googlesheet

import (
	"context"
	"easycodeapp/pkg/logger/interfaces"
	"easycodeapp/pkg/request"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Config struct {
	BufferSize         int
	RequestUpdatePause int
	// Logger определяет способ логирования:
	// - nil: будет использован стандартный log.Logger
	// - false: логирование будет отключено
	// - interfaces.BasicLogger: будет использован базовый логгер
	// - interfaces.SimpleLogger: будет использован расширенный логгер
	Logger          interface{}
	CredentialsFile string
}

// GoogleSheets - структуру для работы с Google таблицами
type GoogleSheets struct {
	*sheets.Service                         // Сервис Google Sheets API
	Request         *request.RequestHandler // Структура для откладывания выполнений функций

	// Logger определяет способ логирования:
	// - nil: будет использован стандартный log.Logger
	// - false: логирование будет отключено
	// - interfaces.BasicLogger: будет использован базовый логгер
	// - interfaces.SimpleLogger: будет использован расширенный логгер
	logger interface{}

	loggingEnabled bool
}

// log записывает сообщение в лог, если логирование включено.
// Использует разные методы в зависимости от типа логгера.
func (app *GoogleSheets) log(msg string) {
	if app == nil {
		fmt.Println("GoogleSheets is not initialized")
		return
	}

	if !app.loggingEnabled {
		return
	}

	switch l := app.logger.(type) {
	case interfaces.SimpleLogger:
		l.Info(msg)
	case interfaces.BasicLogger:
		l.Print(msg)
	}
}

// logf записывает форматированное сообщение в лог, если логирование включено.
// Использует разные методы в зависимости от типа логгера.
func (app *GoogleSheets) logf(format string, args ...interface{}) {
	if app == nil {
		fmt.Println("GoogleSheets is not initialized")
		return
	}

	if !app.loggingEnabled {
		return
	}

	switch l := app.logger.(type) {
	case interfaces.SimpleLogger:
		l.Infof(format, args...)
	case interfaces.BasicLogger:
		l.Printf(format, args...)
	}
}

func (app *GoogleSheets) logError(format string, args ...interface{}) {
	if app == nil {
		fmt.Println("GoogleSheets is not initialized")
		return
	}

	if !app.loggingEnabled {
		return
	}

	switch l := app.logger.(type) {
	case interfaces.SimpleLogger:
		l.Errorf(format, args...)
	case interfaces.BasicLogger:
		l.Printf("ERROR: "+format, args...)
	}
}

// NewGoogleSheets создает новый экземпляр GoogleSheets
func NewGoogleSheets(config Config) (*GoogleSheets, error) {
	Request, err := request.NewRequestHandler(request.Config{
		BufferSize: config.BufferSize,
		Logger:     config.Logger,
	})
	if err != nil {
		log.Printf("Ошибка инициализации RequestHandler: %v", err)
		return nil, err
	}

	ctx := context.Background()
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(config.CredentialsFile),
		option.WithScopes(sheets.SpreadsheetsScope))
	if err != nil {
		return nil, fmt.Errorf("не удается инициализировать сервис Google Sheets: %v", err)
	}

	app := &GoogleSheets{
		Request:        Request,
		Service:        service,
		logger:         config.Logger,
		loggingEnabled: true,
	}

	// Настройка логгера
	if v, ok := config.Logger.(bool); ok && !v {
		app.loggingEnabled = false
	} else if config.Logger == nil {
		app.logger = log.New(os.Stdout, "request: ", log.LstdFlags)
	} else if l, ok := config.Logger.(interfaces.BasicLogger); ok {
		app.logger = l
	} else if l, ok := config.Logger.(interfaces.SimpleLogger); ok {
		app.logger = l
	} else {
		return nil, errors.New("неподдерживаемый тип логгера")
	}

	pause := time.Duration(config.RequestUpdatePause) * time.Second
	go app.Request.ProcessRequestsWithDynamicPause(pause, request.IncrementPause(1.5, 30*time.Second))

	return app, nil
}

// fetchValueRangeWithWaitGroup выполняет запрос к Google Sheets API с использованием sync.WaitGroup
func (app *GoogleSheets) FetchValueRangeWithWaitGroup(sheetID, readRange string) (*sheets.ValueRange, error) {
	var err error
	var resp *sheets.ValueRange
	var wg sync.WaitGroup
	wg.Add(1)
	app.log("FetchValueRangeWithWaitGroup: " + fmt.Sprint(app.Request))
	app.Request.HandleRequest(func() error {
		defer wg.Done()
		// Проверяем, что передан ID, а не название таблицы
		if !strings.HasPrefix(sheetID, "1") || len(sheetID) < 20 {
			return fmt.Errorf("некорректный ID таблицы: %s", sheetID)
		}
		resp, err = app.Service.Spreadsheets.Values.Get(sheetID, readRange).Do()
		if err != nil {
			return fmt.Errorf("не удалось извлечь данные из таблицы: %v", err)
		}
		return nil
	})

	wg.Wait()
	return resp, err
}

// GetCellValue получает значение из указанной ячейки Google таблицы
// cell - в формате "C1"
func (app *GoogleSheets) GetCellValue(sheetID, sheetName, cell string) (string, error) {
	readRange := fmt.Sprintf("%s!%s", sheetName, cell)
	app.log("Получение значения из таблицы " + sheetID + " ; листа: " + sheetName + " ; ячейка: " + cell)

	resp, err := app.FetchValueRangeWithWaitGroup(sheetID, readRange)
	if err != nil {
		app.logError("Ошибка при выполнении запроса: " + err.Error())
		return "", err
	}

	if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
		return "попытка взять значение ячейки вернуло нулевой ответ", nil
	}

	value := fmt.Sprintf("%v", resp.Values[0][0])
	app.log("Получено значение: " + value)
	return value, nil
}

// GetColumnValues получает значения из указанного диапазона столбца Google таблицы
// columnRange диапазон в формате "C2:C239"
func (app *GoogleSheets) GetColumnValues(sheetID, sheetName, columnRange string) ([]string, error) {
	readRange := fmt.Sprintf("%s!%s", sheetName, columnRange)
	app.log("Получение значения из таблицы " + sheetID + " ; листа: " + sheetName + " ; столбца: " + columnRange)

	resp, err := app.FetchValueRangeWithWaitGroup(sheetID, readRange)
	if err != nil {
		return []string{}, err // Возвращаем пустой слайс вместо nil
	}

	// Добавляем проверку на nil
	if resp == nil {
		return []string{}, fmt.Errorf("получен пустой ответ")
	}

	var values []string
	for _, row := range resp.Values {
		if len(row) > 0 {
			value := fmt.Sprintf("%v", row[0])
			values = append(values, value)
		} else {
			values = append(values, "")
		}
	}

	return values, nil
}

// GetMatrix получает значения из указанного диапазона в виде матрицы
// startCol, endCol - номера столбцов (A=1, B=2, etc.) x1, x2
// startRow, endRow - номера строк y1, y2
// GetMatrix получает значения из указанного диапазона в виде матрицы
func (app *GoogleSheets) GetMatrix(sheetID, sheetName string, startCol, endCol, startRow, endRow int) ([][]string, error) {
	if startCol < 1 || endCol > 26 || startCol > endCol {
		return nil, fmt.Errorf("некорректные номера столбцов: начало=%d, конец=%d (допустимо от 1 до 26)", startCol, endCol)
	}
	if startRow < 1 || startRow > endRow {
		return nil, fmt.Errorf("некорректные номера строк: начало=%d, конец=%d", startRow, endRow)
	}

	startColLetter := string(rune('A' + startCol - 1))
	endColLetter := string(rune('A' + endCol - 1))

	dataRange := fmt.Sprintf("%s%d:%s%d", startColLetter, startRow, endColLetter, endRow)
	readRange := fmt.Sprintf("%s!%s", sheetName, dataRange)

	app.log("Получение матрицы из таблицы " + sheetID + " ; листа: " + sheetName +
		" ; диапазон: " + dataRange)

	resp, err := app.FetchValueRangeWithWaitGroup(sheetID, readRange)
	if err != nil {
		return [][]string{}, err
	}

	if resp == nil {
		return [][]string{}, fmt.Errorf("получен пустой ответ")
	}

	numRows := endRow - startRow + 1
	numCols := endCol - startCol + 1

	// Создаем транспонированную матрицу
	// Теперь первый индекс - столбец, второй - строка
	matrix := make([][]string, numCols)
	for i := range matrix {
		matrix[i] = make([]string, numRows)
		for j := range matrix[i] {
			matrix[i][j] = "" // Заполняем пустыми строками по умолчанию
		}
	}

	// Заполняем транспонированную матрицу
	for rowIdx, row := range resp.Values {
		if rowIdx >= numRows {
			break
		}
		for colIdx := 0; colIdx < numCols && colIdx < len(row); colIdx++ {
			if row[colIdx] != nil {
				// Записываем в транспонированную матрицу
				matrix[colIdx][rowIdx] = fmt.Sprintf("%v", row[colIdx])
			}
		}
	}

	// app.log("Получена матрица: ", matrix)

	return matrix, nil
}

// setNewLineсинхронно заносит в строку таблицы новые данные
func (app *GoogleSheets) SetLine(sheetId, listName, start, finish string, data []string, skip, index int) error {
	interfaceDataBefore := make([][]interface{}, 1)
	for i := range interfaceDataBefore {
		interfaceDataBefore[i] = make([]interface{}, len(data))
		for j := range interfaceDataBefore[i] {
			interfaceDataBefore[i][j] = data[skip+j]
		}
	}
	valueRange := &sheets.ValueRange{
		Values: interfaceDataBefore,
	}

	insertRange := fmt.Sprintf("%s!%s%d:%s%d", listName, start, index, finish, index)

	var wg sync.WaitGroup
	wg.Add(1)
	var err error

	app.Request.HandleRequest(func() error {
		defer wg.Done()
		_, err = app.Spreadsheets.Values.Update(sheetId, insertRange, valueRange).ValueInputOption("USER_ENTERED").Do()
		if err != nil {
			return errors.New("не удалось обновить данные в электронной таблице: " + err.Error())
		}
		return err
	})
	wg.Wait()
	if err != nil {
		return err
	}

	return nil
}

// readColumnValuesRange синхронная
func (app *GoogleSheets) ReadColumnValuesRange(sheetID, listName, columnRange string) ([]string, error) {
	readRange := fmt.Sprintf("%s!%s", listName, columnRange)

	var err error
	var resp *sheets.ValueRange
	var wg sync.WaitGroup
	wg.Add(1)

	app.Request.HandleRequest(func() error {
		defer wg.Done()
		resp, err = app.Service.Spreadsheets.Values.Get(sheetID, readRange).Do()
		if err != nil {
			return fmt.Errorf("не удалось извлечь данные из таблицы: %v", err)
		}
		return err
	})
	wg.Wait()
	if err != nil {
		return nil, err
	}

	var values []string
	for _, row := range resp.Values {
		if len(row) > 0 {
			switch v := row[0].(type) {
			case string:
				values = append(values, v)
			default:
				// Если значение не строкового типа, конвертируем его в строку
				values = append(values, fmt.Sprintf("%v", v))
			}
		} else {
			// Если ячейка пуста, добавляем пустую строку
			values = append(values, "")
		}
	}
	return values, nil
}

// ReadColumnValues синхроно считывает данные из столбца
// SheetID - id google таблицы, sheetName - имя листа, column - колона (следует указывать A:A или A1:A, A2:A, если надо пропусти первый)
func (app *GoogleSheets) ReadColumnValues(sheetID, listName, column string) ([]string, error) {
	var values []string
	var err error
	var wg sync.WaitGroup
	wg.Add(1)

	app.Request.HandleRequest(func() error {
		defer wg.Done()
		readRange := fmt.Sprintf("%s!%s", listName, column)
		var resp *sheets.ValueRange
		resp, err = app.Service.Spreadsheets.Values.Get(sheetID, readRange).Do()
		if err != nil {
			return fmt.Errorf("не удалось извлечь данные из таблицы: %v", err)
		}
		fmt.Println("Запрос вернул: ", resp)
		for _, row := range resp.Values {
			var rowData []string
			for _, cell := range row {
				cellValue, ok := cell.(string)
				if !ok {
					return fmt.Errorf("неожиданный тип значения в таблице")
				}
				rowData = append(rowData, cellValue)
			}
			if len(rowData) > 0 {
				values = append(values, rowData[0])
			}
		}
		return nil
	})

	wg.Wait()
	return values, err
}

// getEmptyIndex синхронно получает последний свободный номер строки
// columns - имена столбцов ([]string{"A","B","D","L"})
func (app *GoogleSheets) GetEmptyIndex(sheetID, listName string, columns []string) (int, error) {
	if len(columns) == 0 {
		return 0, errors.New("список столбцов пуст")
	}

	firstColumn := columns[0]
	lastColumn := columns[len(columns)-1]
	fullRange := fmt.Sprintf("%s!%s:%s", listName, firstColumn, lastColumn)

	app.log("Запрос данных из таблицы range: " + fullRange)

	var err error
	var resp *sheets.ValueRange
	var wg sync.WaitGroup
	wg.Add(1)

	app.Request.HandleRequest(func() error {
		defer wg.Done()
		resp, err = app.Spreadsheets.Values.Get(sheetID, fullRange).Do()
		if err != nil {
			app.logError("Ошибка при получении данных из таблицы: " + err.Error())
			return fmt.Errorf("не удалось извлечь данные из таблицы: %v", err)
		}
		return nil
	})
	wg.Wait()
	if err != nil {
		return 0, err
	}

	app.log("Получены данные из таблицы: " + strconv.Itoa(len(resp.Values)))

	for i := len(resp.Values) - 1; i >= 0; i-- {
		rowIsEmpty := true
		for j := 0; j < len(columns) && j < len(resp.Values[i]); j++ {
			if strings.TrimSpace(fmt.Sprintf("%v", resp.Values[i][j])) != "" {
				rowIsEmpty = false
				app.log("Столбец не пустой: " + columns[j] + " в строке " + strconv.Itoa(i+2))
				break
			}
		}
		if !rowIsEmpty {
			emptyIndex := i + 2
			app.log("Найдена первая непустая строка " + " index " + strconv.Itoa(i) + " emptyIndex " + strconv.Itoa(emptyIndex))
			return emptyIndex, nil
		}
	}

	app.log("Все строки пусты, возвращаем первую строку")
	return 1, nil
}
