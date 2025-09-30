package repository

import "errors"

var (
	ErrFileNotFound       = errors.New("FILE NOT FOUND")
	ErrInvalidFileID      = errors.New("INVALID FILE ID")
	ErrFileTooLarge       = errors.New("FILE TOO LARGE")
	ErrInvalidFilename    = errors.New("INVALIDFILENAME")
	ErrStorageUnavailable = errors.New("STORAGE UNAVAILABLE")
	ErrFileIsEmpty        = errors.New("FILE IS EMPTY")
	ErrFailToDeleteFile   = errors.New("FAIL TO DELETE FILE")
)
