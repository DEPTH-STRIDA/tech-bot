package web

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"text/template"
	"time"
)

// NewTemplateCache находит файлы шаблонов и создает карту маршрутов
func NewTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}
	files, err := filepath.Glob(filepath.Join(dir, "*.tmpl"))
	if err != nil {
		return nil, err
	}

	log.Printf("Найдено %d файлов шаблонов в директории %s", len(files), dir)

	for _, file := range files {
		log.Printf("Обработка файла: %s", file)
		match, err := filepath.Match("*.page.tmpl", filepath.Base(file))
		if err != nil {
			return nil, err
		}
		if match {
			name := filepath.Base(file)
			ts, err := template.ParseFiles(file)
			if err != nil {
				log.Printf("Ошибка при парсинге файла %s: %v", file, err)
				return nil, err
			}
			cache[name] = ts
			log.Printf("Шаблон %s успешно загружен", name)
		} else {
			log.Printf("Файл %s не соответствует шаблону *.page.tmpl", file)
		}
	}
	return cache, nil
}

func (app *WebApp) render(w http.ResponseWriter, name string, data map[string]interface{}) error {
	ts, ok := app.TemplateCache[name]
	if !ok {
		return errors.New("не удалось использовать шаблон. Шаблон " + name + " не существует")
	}

	// Если data не предоставлена, создаем пустую map
	if data == nil {
		data = make(map[string]interface{})
	}

	// Добавляем версию, если она еще не установлена
	if _, exists := data["Version"]; !exists {
		data["Version"] = fmt.Sprintf("%d", time.Now().Unix())
	}

	err := ts.Execute(w, data)
	if err != nil {
		return err
	}
	return nil
}
