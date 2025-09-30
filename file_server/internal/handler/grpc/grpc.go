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

type Handler struct {
	gen.UnimplementedFileServiceServer
	ctrl *file.Controller
}

func NewGrpc(ctrl *file.Controller) *Handler {
	return &Handler{
		ctrl: ctrl,
	}
}

// Upload file request
func (h *Handler) UploadFile(ctx context.Context, req *gen.UploadFileRequest) (*gen.UploadFileResponse, error) {
	// validating request
	if req.Filename == "" {
		return nil, status.Error(codes.InvalidArgument, "filename is required")
	}

	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	// internal request
	uploadReq := &model.UploadRequest{
		Filename: req.Filename,
		Data:     req.Data,
	}

	// calling controller
	resp, err := h.ctrl.UploadFile(ctx, uploadReq)
	if err != nil {
		return nil, h.handleError(err)
	}
	return &gen.UploadFileResponse{
		FileId: resp.FileID,
	}, nil
}

func (h *Handler) GetFile(ctx context.Context, req *gen.GetFileRequest) (*gen.GetFileResponse, error) {
	// validating request
	if req.FileId == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	// internal request
	getReq := &model.GetRequest{
		FileID: req.FileId,
	}

	// calling controller
	resp, err := h.ctrl.GetFile(ctx, getReq)
	if err != nil {
		return nil, h.handleError(err)
	}
	return &gen.GetFileResponse{
		Filname: resp.Filename,
		Data:    resp.Data,
	}, nil
}

func (h *Handler) ListFiles(ctx context.Context, req *gen.ListFilesRequest) (*gen.ListFilesResponse, error) {
	// calling controller
	resp, err := h.ctrl.ListFiles(ctx)
	if err != nil {
		return nil, h.handleError(err)
	}

	// formatting to grpc
	files := make([]*gen.FileInfo, 0, len(resp.Files))
	for _, file := range resp.Files {
		files = append(files, &gen.FileInfo{
			FileId:    file.ID,
			Filename:  file.Filename,
			CreatedAt: file.CreatedAt.Unix(),
			UpdatedAt: file.UpdatedAt.Unix(),
		})
	}

	return &gen.ListFilesResponse{
		Files: files,
	}, nil
}

func (h *Handler) handleError(err error) error {
	switch err {
	case repository.ErrFileNotFound:
		return status.Error(codes.NotFound, "FILE NOT FOUND")
	case repository.ErrInvalidFileID:
		return status.Error(codes.InvalidArgument, "INVALID FILE ID")
	case repository.ErrFileTooLarge:
		return status.Error(codes.InvalidArgument, "FILE IS TOO LARGE")
	case repository.ErrInvalidFilename:
		return status.Error(codes.InvalidArgument, "INVALID FILENAME")
	case repository.ErrStorageUnavailable:
		return status.Error(codes.Internal, "STORAGE UNAVAILABLE")
	default:
		return status.Error(codes.Internal, fmt.Sprintf("INTERNAL ERROR: %v", err))
	}
}
