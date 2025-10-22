// file.go - модели данных для файлового сервиса
// Определяет структуры данных для работы с файлами на всех слоях приложения
package model

import "time"

// FileInfo содержит метаданные файла
// Используется для хранения информации о файле без его содержимого
type FileInfo struct {
	ID        string    `json:"id"`         // Уникальный идентификатор файла (MD5 хэш содержимого)
	Filename  string    `json:"filename"`   // Оригинальное имя файла
	CreatedAt time.Time `json:"created_at"` // Время создания файла
	UpdatedAt time.Time `json:"updated_at"` // Время последнего обновления файла
	Size      int64     `json:"size"`       // Размер файла в байтах
}

// File содержит полную информацию о файле включая содержимое
// Используется для передачи файла с метаданными
type File struct {
	Info FileInfo `json:"info"` // Метаданные файла
	Data []byte   `json:"data"` // Содержимое файла в байтах
}

// UploadRequest представляет запрос на загрузку файла
// Содержит имя файла и его содержимое
type UploadRequest struct {
	Filename string // Имя загружаемого файла
	Data     []byte // Содержимое файла в байтах
}

// UploadResponse представляет ответ на запрос загрузки файла
// Содержит уникальный идентификатор сохраненного файла
type UploadResponse struct {
	FileID string // Уникальный идентификатор сохраненного файла
}

// GetRequest представляет запрос на получение файла
// Содержит идентификатор файла для загрузки
type GetRequest struct {
	FileID string // Идентификатор файла для загрузки
}

// GetResponse представляет ответ на запрос получения файла
// Содержит имя файла и его содержимое
type GetResponse struct {
	Filename string // Имя файла
	Data     []byte // Содержимое файла в байтах
}

// ListResponse представляет ответ на запрос списка файлов
// Содержит массив метаданных всех файлов
type ListResponse struct {
	Files []FileInfo // Массив метаданных файлов
}
