package handler

import (
	"encoding/json"
	"net/http"

	"myApp/internal/core"
)

// ProductJSONView — структура, которая будет сериализована в JSON.
// Экспортируемые поля и теги `json` — чтобы результат был удобочитаемым.
type ProductJSONView struct {
	Name        string            `json:"name"`
	Article     string            `json:"article"`
	Price       string            `json:"price"`
	ImageURL    string            `json:"image_url"`
	Short       string            `json:"short"`
	Description string            `json:"description"`
	BackURL     string            `json:"back_url"`
	Attributes  map[string]string `json:"attributes"`
}

// ProductJSON — возвращает JSON ответа о товаре (GET /api/product/{sku})
func ProductJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Пример данных — в реальном проекте извлечём из БД по SKU/ID
		data := ProductJSONView{
			Name:        "Товар №1",
			Article:     "ART-0001",
			Price:       "29.90",
			ImageURL:    "https://picsum.photos/seed/p1/800/800",
			Short:       "Увлажняющее средство для волос...",
			Description: "Лёгкая текстура и свежий аромат. Подходит для ежедневного применения.",
			BackURL:     "/",
			Attributes: map[string]string{
				"brand":    "BrandName",
				"country":  "Finland",
				"volume":   "250 ml",
				"category": "hair-care",
			},
		}

		// Заголовки
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store") // по умолчанию — не кэшировать

		// Сериализация в JSON и отправка.
		// Используем Encoder, чтобы избежать полного буфера в памяти для больших данных.
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(true) // защита: экранируем <,>,& (по умолчанию true)
		//enc.SetIndent("", "  ") // если хочешь "красивый" JSON в dev — раскомментируй

		if err := enc.Encode(data); err != nil {
			core.LogError("Ошибка кодирования JSON", map[string]interface{}{
				"error": err.Error(),
				"path":  r.URL.Path,
			})
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}
