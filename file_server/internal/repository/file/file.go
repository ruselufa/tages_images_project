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

type Repository struct {
	storagePath string
	mutex       sync.RWMutex
	files       map[string]*model.FileInfo
}

func NewRepo(storagePath string) (*Repository, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("FAILED TO CREATE STORAGE DIRECTORY: %w", err)
	}

	repo := &Repository{
		storagePath: storagePath,
		files:       make(map[string]*model.FileInfo),
	}
	// loadExistingFiles in cache
	if err := repo.loadExistingFiles(); err != nil {
		return nil, fmt.Errorf("FAILED TO LOAD EXISTING FILES: %w", err)
	}

	return repo, nil
}

func (r *Repository) loadExistingFiles() error {
	entities, err := os.ReadDir(r.storagePath)
	if err != nil {
		return err
	}

	for _, entry := range entities {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileInfo := &model.FileInfo{
			ID:        entry.Name(),
			Filename:  entry.Name(),
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
			Size:      info.Size(),
		}

		r.files[entry.Name()] = fileInfo
	}

	return nil
}

func (r *Repository) SaveFile(filename string, data []byte) (string, error) {
	// validation incoming data
	if err := r.validateFile(filename, data); err != nil {
		return "", err
	}

	// generate unique Id
	hash := md5.Sum(data)
	fileID := hex.EncodeToString(hash[:])

	// checking file for existing one
	r.mutex.Lock()
	if _, exists := r.files[fileID]; exists {
		r.mutex.RUnlock()
		return fileID, nil
	}
	r.mutex.RUnlock()

	// saving file to disk
	filePath := filepath.Join(r.storagePath, fileID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("FAILED TO WRITE FILE: %w", err)
	}

	// updating metadata cache
	now := time.Now()
	fileInfo := &model.FileInfo{
		ID:        fileID,
		Filename:  filename,
		CreatedAt: now,
		UpdatedAt: now,
		Size:      int64(len(data)),
	}

	r.mutex.Lock()
	r.files[fileID] = fileInfo
	r.mutex.Unlock()

	return fileID, nil
}

// getfile by it ID
func (r *Repository) GetFile(fileID string) (*model.File, error) {
	if fileID == "" {
		return nil, repository.ErrInvalidFileID
	}

	// checking metadata cache
	r.mutex.RLock()
	fileInfo, exists := r.files[fileID]
	r.mutex.RUnlock()

	if !exists {
		return nil, repository.ErrFileNotFound
	}

	// reading file from disk
	filePath := filepath.Join(r.storagePath, fileID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// file was deleted fr disk, but exists in cache
			r.mutex.Lock()
			delete(r.files, fileID)
			r.mutex.Unlock()
			return nil, repository.ErrFileNotFound
		}
		return nil, fmt.Errorf("FAILED TO READ FILE: %w", err)
	}

	return &model.File{
		Info: *fileInfo,
		Data: data,
	}, nil
}

func (r *Repository) ListFiles() ([]model.FileInfo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	files := make([]model.FileInfo, 0, len(r.files))
	for _, fileInfo := range r.files {
		files = append(files, *fileInfo)
	}

	return files, nil
}

func (r *Repository) validateFile(filename string, data []byte) error {
	// checking file type
	if strings.TrimSpace(filename) == "" {
		return repository.ErrInvalidFilename
	}

	// checking filesize, max 10MB
	const maxFileSize = 10 * 1024 * 1024
	if len(data) > maxFileSize {
		return repository.ErrFileTooLarge
	}

	// checking file is not empty
	if len(data) == 0 {
		return repository.ErrFileIsEmpty
	}

	return nil
}

func (r *Repository) GetFileInfo(fileID string) (*model.FileInfo, error) {
	if fileID == "" {
		return nil, repository.ErrInvalidFileID
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	fileInfo, exists := r.files[fileID]
	if !exists {
		return nil, repository.ErrFileNotFound
	}

	return fileInfo, nil
}

func (r *Repository) DeleteFile(fileID string) error {
	if fileID == "" {
		return repository.ErrInvalidFileID
	}

	// deleting file from disk
	filePath := filepath.Join(r.storagePath, fileID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return repository.ErrFailToDeleteFile
	}

	// deleting from cache
	r.mutex.Lock()
	delete(r.files, fileID)
	r.mutex.Unlock()

	return nil
}

func (r *Repository) GetStats() (int, int64, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	fileCount := len(r.files)
	totalSize := int64(0)

	for _, fileInfo := range r.files {
		totalSize += fileInfo.Size
	}

	return fileCount, totalSize, nil
}
