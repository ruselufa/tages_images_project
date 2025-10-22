package grpc

import (
	"context"
	"file_server/gen"
	"file_server/internal/controller/file"
	"file_server/internal/repository"
	"file_server/pkg/model"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler - gRPC обработчик для файлового сервиса
// Реализует интерфейс FileServiceServer из сгенерированного protobuf кода
// Служит мостом между gRPC запросами и внутренней бизнес-логикой
type Handler struct {
	gen.UnimplementedFileServiceServer                  // Встраиваем базовую реализацию для совместимости
	ctrl                               *file.Controller // Контроллер для обработки бизнес-логики
}

// NewGrpc создает новый экземпляр gRPC обработчика
// Принимает контроллер для обработки бизнес-логики файловых операций
func NewGrpc(ctrl *file.Controller) *Handler {
	return &Handler{
		ctrl: ctrl,
	}
}

// UploadFile обрабатывает gRPC запрос на загрузку файла
// Валидирует входные данные, преобразует в внутренний формат и делегирует контроллеру
func (h *Handler) UploadFile(ctx context.Context, req *gen.UploadFileRequest) (*gen.UploadFileResponse, error) {
	// Валидация входных данных gRPC запроса
	if req.Filename == "" {
		return nil, status.Error(codes.InvalidArgument, "filename is required")
	}

	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	// Преобразование gRPC запроса в внутреннюю модель приложения
	uploadReq := &model.UploadRequest{
		Filename: req.Filename,
		Data:     req.Data,
	}

	// Делегирование обработки контроллеру (бизнес-логика)
	resp, err := h.ctrl.UploadFile(ctx, uploadReq)
	if err != nil {
		return nil, h.handleError(err) // Преобразование внутренних ошибок в gRPC статусы
	}

	// Преобразование ответа контроллера в gRPC формат
	return &gen.UploadFileResponse{
		FileId: resp.FileID,
	}, nil
}

// GetFile обрабатывает gRPC запрос на получение файла по ID
// Валидирует входные данные, преобразует в внутренний формат и делегирует контроллеру
func (h *Handler) GetFile(ctx context.Context, req *gen.GetFileRequest) (*gen.GetFileResponse, error) {
	// Валидация входных данных gRPC запроса
	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// Преобразование gRPC запроса в внутреннюю модель приложения
	getReq := &model.GetRequest{
		FileID: req.FileId,
	}

	// Делегирование обработки контроллеру (бизнес-логика)
	resp, err := h.ctrl.GetFile(ctx, getReq)
	if err != nil {
		return nil, h.handleError(err) // Преобразование внутренних ошибок в gRPC статусы
	}

	// Преобразование ответа контроллера в gRPC формат
	return &gen.GetFileResponse{
		Filname: resp.Filename,
		Data:    resp.Data,
	}, nil
}

// ListFiles обрабатывает gRPC запрос на получение списка всех файлов
// Делегирует контроллеру и преобразует результат в gRPC формат
func (h *Handler) ListFiles(ctx context.Context, req *gen.ListFilesRequest) (*gen.ListFilesResponse, error) {
	// Делегирование обработки контроллеру (бизнес-логика)
	// ListFiles не требует валидации входных параметров, так как запрос пустой
	resp, err := h.ctrl.ListFiles(ctx)
	if err != nil {
		return nil, h.handleError(err) // Преобразование внутренних ошибок в gRPC статусы
	}

	// Преобразование внутренних моделей файлов в gRPC формат
	files := make([]*gen.FileInfo, 0, len(resp.Files))
	for _, file := range resp.Files {
		files = append(files, &gen.FileInfo{
			FileId:    file.ID,
			Filename:  file.Filename,
			CreatedAt: file.CreatedAt.Unix(), // Преобразование времени в Unix timestamp
			UpdatedAt: file.UpdatedAt.Unix(), // Преобразование времени в Unix timestamp
		})
	}

	// Возврат gRPC ответа со списком файлов
	return &gen.ListFilesResponse{
		Files: files,
	}, nil
}

// handleError преобразует внутренние ошибки приложения в gRPC статусы
// Обеспечивает единообразную обработку ошибок на уровне gRPC API
func (h *Handler) handleError(err error) error {
	switch err {
	// Файл не найден в хранилище
	case repository.ErrFileNotFound:
		return status.Error(codes.NotFound, "FILE NOT FOUND")

	// Некорректный формат ID файла
	case repository.ErrInvalidFileID:
		return status.Error(codes.InvalidArgument, "INVALID FILE ID")

	// Файл превышает максимально допустимый размер
	case repository.ErrFileTooLarge:
		return status.Error(codes.InvalidArgument, "FILE IS TOO LARGE")

	// Некорректное имя файла (пустое, содержит недопустимые символы)
	case repository.ErrInvalidFilename:
		return status.Error(codes.InvalidArgument, "INVALID FILENAME")

	// Проблемы с доступом к хранилищу файлов
	case repository.ErrStorageUnavailable:
		return status.Error(codes.Internal, "STORAGE UNAVAILABLE")

	// Неизвестные ошибки - возвращаем как внутренние ошибки сервера
	default:
		return status.Error(codes.Internal, fmt.Sprintf("INTERNAL ERROR: %v", err))
	}
}
