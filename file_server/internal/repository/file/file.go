// file.go - репозиторий для работы с файлами
// Обеспечивает сохранение, загрузку и управление файлами на диске
// Использует кэш метаданных для быстрого доступа к информации о файлах
package file

import (
	"crypto/md5"
	"encoding/hex"
	"file_server/internal/repository"
	"file_server/pkg/model"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Repository - репозиторий для работы с файлами
// Хранит файлы на диске и кэширует их метаданные в памяти
type Repository struct {
	storagePath string                     // Путь к директории хранения файлов
	mutex       sync.RWMutex               // Мьютекс для thread-safe доступа к кэшу
	files       map[string]*model.FileInfo // Кэш метаданных файлов (ID -> FileInfo)
}

// NewRepo создает новый экземпляр репозитория
// Создает директорию хранения и загружает существующие файлы в кэш
func NewRepo(storagePath string) (*Repository, error) {
	// Создание директории хранения файлов (если не существует)
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("FAILED TO CREATE STORAGE DIRECTORY: %w", err)
	}

	// Создание экземпляра репозитория
	repo := &Repository{
		storagePath: storagePath,
		files:       make(map[string]*model.FileInfo), // Инициализация кэша метаданных
	}

	// Загрузка существующих файлов в кэш при инициализации
	if err := repo.loadExistingFiles(); err != nil {
		return nil, fmt.Errorf("FAILED TO LOAD EXISTING FILES: %w", err)
	}

	return repo, nil
}

// loadExistingFiles загружает информацию о существующих файлах в кэш
// Сканирует директорию хранения и создает метаданные для каждого файла
func (r *Repository) loadExistingFiles() error {
	// Чтение содержимого директории хранения
	entities, err := os.ReadDir(r.storagePath)
	if err != nil {
		return err
	}

	// Обработка каждого элемента в директории
	for _, entry := range entities {
		// Пропускаем поддиректории
		if entry.IsDir() {
			continue
		}

		// Получение информации о файле
		info, err := entry.Info()
		if err != nil {
			continue // Пропускаем файлы с ошибками доступа
		}

		// Создание метаданных файла
		// ID файла = имя файла (MD5 хэш содержимого)
		fileInfo := &model.FileInfo{
			ID:        entry.Name(),   // ID файла (MD5 хэш)
			Filename:  entry.Name(),   // Имя файла (временно = ID, будет обновлено при загрузке)
			CreatedAt: info.ModTime(), // Время создания (время модификации файла)
			UpdatedAt: info.ModTime(), // Время обновления (время модификации файла)
			Size:      info.Size(),    // Размер файла в байтах
		}

		// Добавление метаданных в кэш
		r.files[entry.Name()] = fileInfo
	}

	return nil
}

// SaveFile сохраняет файл на диск и обновляет кэш метаданных
// Использует MD5 хэш содержимого как уникальный ID файла
func (r *Repository) SaveFile(filename string, data []byte) (string, error) {
	// Валидация входящих данных (имя файла, размер, содержимое)
	if err := r.validateFile(filename, data); err != nil {
		return "", err
	}

	// Генерация уникального ID на основе MD5 хэша содержимого файла
	hash := md5.Sum(data)
	fileID := hex.EncodeToString(hash[:])

	// Проверка, существует ли файл с таким содержимым (дедупликация)
	r.mutex.RLock()
	if _, exists := r.files[fileID]; exists {
		r.mutex.RUnlock()
		return fileID, nil // Возвращаем существующий ID без сохранения
	}
	r.mutex.RUnlock()

	// Сохранение файла на диск
	filePath := filepath.Join(r.storagePath, fileID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("FAILED TO WRITE FILE: %w", err)
	}

	// Создание метаданных файла
	now := time.Now()
	fileInfo := &model.FileInfo{
		ID:        fileID,           // Уникальный ID (MD5 хэш)
		Filename:  filename,         // Оригинальное имя файла
		CreatedAt: now,              // Время создания
		UpdatedAt: now,              // Время обновления
		Size:      int64(len(data)), // Размер файла в байтах
	}

	// Обновление кэша метаданных
	r.mutex.Lock()
	r.files[fileID] = fileInfo
	r.mutex.Unlock()

	return fileID, nil
}

// GetFile загружает файл по его ID
// Проверяет кэш метаданных и читает содержимое с диска
func (r *Repository) GetFile(fileID string) (*model.File, error) {
	// Валидация ID файла
	if fileID == "" {
		return nil, repository.ErrInvalidFileID
	}

	// Проверка существования файла в кэше метаданных
	r.mutex.RLock()
	fileInfo, exists := r.files[fileID]
	r.mutex.RUnlock()

	if !exists {
		return nil, repository.ErrFileNotFound
	}

	// Чтение содержимого файла с диска
	filePath := filepath.Join(r.storagePath, fileID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл был удален с диска, но существует в кэше - синхронизируем кэш
			r.mutex.Lock()
			delete(r.files, fileID)
			r.mutex.Unlock()
			return nil, repository.ErrFileNotFound
		}
		return nil, fmt.Errorf("FAILED TO READ FILE: %w", err)
	}

	// Возврат файла с метаданными и содержимым
	return &model.File{
		Info: *fileInfo,
		Data: data,
	}, nil
}

// ListFiles возвращает список всех файлов из кэша метаданных
// Создает копию метаданных для безопасного возврата
func (r *Repository) ListFiles() ([]model.FileInfo, error) {
	// Блокировка для безопасного чтения кэша
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Создание слайса с предварительно выделенной емкостью
	files := make([]model.FileInfo, 0, len(r.files))

	// Копирование метаданных из кэша
	for _, fileInfo := range r.files {
		files = append(files, *fileInfo)
	}

	return files, nil
}

// validateFile валидирует входящие данные файла
// Проверяет имя файла, размер и содержимое
func (r *Repository) validateFile(filename string, data []byte) error {
	// Проверка имени файла - не должно быть пустым или содержать только пробелы
	if strings.TrimSpace(filename) == "" {
		return repository.ErrInvalidFilename
	}

	// Проверка размера файла - максимум 10MB
	const maxFileSize = 10 * 1024 * 1024
	if len(data) > maxFileSize {
		return repository.ErrFileTooLarge
	}

	// Проверка, что файл не пустой
	if len(data) == 0 {
		return repository.ErrFileIsEmpty
	}

	return nil
}

// GetFileInfo возвращает метаданные файла по ID
// Читает информацию из кэша без загрузки содержимого файла
func (r *Repository) GetFileInfo(fileID string) (*model.FileInfo, error) {
	// Валидация ID файла
	if fileID == "" {
		return nil, repository.ErrInvalidFileID
	}

	// Блокировка для безопасного чтения кэша
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Поиск метаданных в кэше
	fileInfo, exists := r.files[fileID]
	if !exists {
		return nil, repository.ErrFileNotFound
	}

	return fileInfo, nil
}

// DeleteFile удаляет файл с диска и из кэша метаданных
// Игнорирует ошибку, если файл уже не существует
func (r *Repository) DeleteFile(fileID string) error {
	// Валидация ID файла
	if fileID == "" {
		return repository.ErrInvalidFileID
	}

	// Удаление файла с диска
	filePath := filepath.Join(r.storagePath, fileID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return repository.ErrFailToDeleteFile
	}

	// Удаление метаданных из кэша
	r.mutex.Lock()
	delete(r.files, fileID)
	r.mutex.Unlock()

	return nil
}

// GetStats возвращает статистику репозитория
// Подсчитывает количество файлов и общий размер всех файлов
func (r *Repository) GetStats() (int, int64, error) {
	// Блокировка для безопасного чтения кэша
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Подсчет количества файлов
	fileCount := len(r.files)

	// Подсчет общего размера всех файлов
	totalSize := int64(0)
	for _, fileInfo := range r.files {
		totalSize += fileInfo.Size
	}

	return fileCount, totalSize, nil
}
