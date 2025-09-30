package file

import (
	"context"
	"file_server/internal/repository/file"
	"file_server/pkg/model"
	"fmt"
)

type Controller struct {
	repo *file.Repository
}

func NewController(repo *file.Repository) *Controller {
	return &Controller{
		repo: repo,
	}
}

func (c *Controller) UploadFile(ctx context.Context, req *model.UploadRequest) (*model.UploadResponse, error) {
	// checking context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fileID, err := c.repo.SaveFile(req.Filename, req.Data)
	if err != nil {
		return nil, fmt.Errorf("FAILED TO SAVE FILE: %w", err)
	}

	return &model.UploadResponse{
		FileID: fileID,
	}, nil
}

func (c *Controller) GetFile(ctx context.Context, req *model.GetRequest) (*model.GetResponse, error) {
	//checking context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	file, err := c.repo.GetFile(req.FileID)
	if err != nil {
		return nil, fmt.Errorf("FAILED TO GET FILE: %w", err)
	}

	return &model.GetResponse{
		Filename: file.Info.Filename,
		Data:     file.Data,
	}, nil
}

func (c *Controller) ListFiles(ctx context.Context) (*model.ListResponse, error) {
	// checking context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	files, err := c.repo.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("FAILED TO FIND FILES: %w", err)
	}

	return &model.ListResponse{
		Files: files,
	}, nil
}

func (c *Controller) GetFileInfo(ctx context.Context, fileID string) (*model.FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return c.repo.GetFileInfo(fileID)
}

func (c *Controller) GetStats(ctx context.Context) (int, int64, error) {
	select {
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	default:
	}

	return c.repo.GetStats()
}
