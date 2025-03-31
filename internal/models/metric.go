package models

// Metric описывет параметры метрики, используемые при обмене между агентом и
// сервером в JSON-формате, а именно:
//   - в качестве запроса в `POST /update`
//   - в качестве запроса и ответа в `POST /value`, причем в запросе
//     `POST /value` заполняются поля ID и MType.
//
// Примечание: в ответе от сервера в Delta передается аккумулированное значение
type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}
