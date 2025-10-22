// controller.go - контроллер для обработки бизнес-логики файлового сервиса
// Служит промежуточным слоем между gRPC обработчиком и репозиторием
package file

import (
	"context"
	"file_server/internal/repository/file"
	"file_server/pkg/model"
	"fmt"
)

// Controller - контроллер для файловых операций
// Координирует работу между gRPC обработчиком и репозиторием
// Добавляет проверки контекста и обработку ошибок
type Controller struct {
	repo *file.Repository // Репозиторий для работы с файлами
}

// NewController создает новый экземпляр контроллера
// Принимает репозиторий для работы с файлами
func NewController(repo *file.Repository) *Controller {
	return &Controller{
		repo: repo,
	}
}

// UploadFile обрабатывает запрос на загрузку файла
// Проверяет контекст и делегирует сохранение репозиторию
func (c *Controller) UploadFile(ctx context.Context, req *model.UploadRequest) (*model.UploadResponse, error) {
	// Проверка контекста на отмену операции
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // Возвращаем ошибку отмены контекста
	default:
	}

	// Делегирование сохранения файла репозиторию
	fileID, err := c.repo.SaveFile(req.Filename, req.Data)
	if err != nil {
		return nil, fmt.Errorf("FAILED TO SAVE FILE: %w", err)
	}

	// Возврат успешного ответа с ID файла
	return &model.UploadResponse{
		FileID: fileID,
	}, nil
}

// GetFile обрабатывает запрос на получение файла по ID
// Проверяет контекст и делегирует загрузку репозиторию
func (c *Controller) GetFile(ctx context.Context, req *model.GetRequest) (*model.GetResponse, error) {
	// Проверка контекста на отмену операции
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // Возвращаем ошибку отмены контекста
	default:
	}

	// Делегирование загрузки файла репозиторию
	file, err := c.repo.GetFile(req.FileID)
	if err != nil {
		return nil, fmt.Errorf("FAILED TO GET FILE: %w", err)
	}

	// Возврат успешного ответа с данными файла
	return &model.GetResponse{
		Filename: file.Info.Filename,
		Data:     file.Data,
	}, nil
}

// ListFiles обрабатывает запрос на получение списка всех файлов
// Проверяет контекст и делегирует получение списка репозиторию
func (c *Controller) ListFiles(ctx context.Context) (*model.ListResponse, error) {
	// Проверка контекста на отмену операции
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // Возвращаем ошибку отмены контекста
	default:
	}

	// Делегирование получения списка файлов репозиторию
	files, err := c.repo.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("FAILED TO FIND FILES: %w", err)
	}

	// Возврат успешного ответа со списком файлов
	return &model.ListResponse{
		Files: files,
	}, nil
}

// GetFileInfo получает метаданные файла по ID
// Проверяет контекст и делегирует запрос репозиторию
func (c *Controller) GetFileInfo(ctx context.Context, fileID string) (*model.FileInfo, error) {
	// Проверка контекста на отмену операции
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // Возвращаем ошибку отмены контекста
	default:
	}

	// Делегирование получения метаданных файла репозиторию
	return c.repo.GetFileInfo(fileID)
}

// GetStats получает статистику репозитория (количество файлов и общий размер)
// Проверяет контекст и делегирует запрос репозиторию
func (c *Controller) GetStats(ctx context.Context) (int, int64, error) {
	// Проверка контекста на отмену операции
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err() // Возвращаем ошибку отмены контекста
	default:
	}

	// Делегирование получения статистики репозиторию
	return c.repo.GetStats()
}
