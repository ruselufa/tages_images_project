package model

import "time"

type FileInfo struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Size      int64     `json:"size"`
}

type File struct {
	Info FileInfo `json:"info"`
	Data []byte `json:"data"`
}

type UploadRequest struct {
	Filename string
	Data []byte
}

type UploadResponse struct {
	FileID string
}

type GetRequest struct {
	FileID string
}

type GetResponse struct {
	Filename string
	Data []byte
}

type ListResponse struct {
	Files []FileInfo
}

