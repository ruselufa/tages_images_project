package file

import (
	"context"
	"file_client/gen"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client gen.FileServiceClient
}

// NewClient creates a new client for connection to file FileServiceClient
func NewClient(addr string) (*Client, error) {
	// Creating new conn w/ timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("FAIL TO CONNECT TO SERVER: %w", err)
	}

	return &Client{
		conn:   conn,
		client: gen.NewFileServiceClient(conn),
	}, nil
}

// UploadFile uploads file into the SERVER
func (c *Client) UploadFile(ctx context.Context, filename string, data []byte) (string, error) {
	// creating ctx w/ timout for UploadFile
	uploadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.UploadFile(uploadCtx, &gen.UploadFileRequest{
		Filename: filename,
		Data:     data,
	})
	if err != nil {
		return "", fmt.Errorf("UPLOAD FAILED: %w", err)
	}
	return resp.FileId, nil
}

// DownloadFile downloads file from SERVER
func (c *Client) DownloadFile(ctx context.Context, fileID string) (*gen.GetFileResponse, error) {
	// creating ctx w/ timout for DownloadFile
	downloadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.GetFile(downloadCtx, &gen.GetFileRequest{
		FileId: fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("DOWNLOAD FAILED: %w", err)
	}
	return resp, nil
}

// ListFiles recieving list of files from SERVER
func (c *Client) ListFiles(ctx context.Context) (*gen.ListFilesResponse, error) {
	// creating ctx w/ timeout for recieve list of files
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.ListFiles(listCtx, &gen.ListFilesRequest{})
	if err != nil {
		return nil, fmt.Errorf("RECIEVING LIST OF FILES FAILED: %w", err)
	}
	return resp, nil
}

// UploadFileFromPath
func (c *Client) UploadFileFromPath(ctx context.Context, filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("FAILED TO READ FILE %s: %w", filePath, err)
	}

	filename := filePath
	if lastSlash := lastIndex(filePath, "/"); lastSlash != -1 {
		filename = filePath[lastSlash+1:]
	}
	if lastSlash := lastIndex(filePath, "\\"); lastSlash != -1 {
		filename = filePath[lastSlash+1:]
	}

	return c.UploadFile(ctx, filename, data)
}

// DownloadFileToPath
func (c *Client) DownloadFileToPath(ctx context.Context, fileId, outputPath string) error {
	resp, err := c.DownloadFile(ctx, fileId)
	if err != nil {
		return err
	}

	// check if outputPath is a dir
	if stat, err := os.Stat(outputPath); err == nil && stat.IsDir() {
		return fmt.Errorf("OUTPUT PATH IS A DIRECTORY: %s", outputPath)
	}

	// create dir if not exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("FAILED TO CREATE DIRECTORY %s: %w", dir, err)
	}

	err = os.WriteFile(outputPath, resp.Data, 0644)
	if err != nil {
		return fmt.Errorf("FAILED TO WRITE FILE TO %s: %w", outputPath, err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.ListFiles(ctx)
	return err
}

func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
