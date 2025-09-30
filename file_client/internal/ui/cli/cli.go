package cli

import (
	"bufio"
	"context"
	"file_client/internal/client/file"
	"fmt"
	"os"
	"strings"
	"time"
)

type CLI struct {
	client  *file.Client
	scanner *bufio.Scanner
}

func New(client *file.Client) *CLI {
	return &CLI{
		client:  client,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

func (c *CLI) Run() error {
	c.printWelcome()
	c.printHelp()

	for {
		fmt.Printf("file-client> ")
		if !c.scanner.Scan() {
			break
		}

		line := strings.TrimSpace(c.scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		switch command {
		case "upload":
			c.handleUpload(args)
		case "download":
			c.handleDownload(args)
		case "list":
			c.handleList()
		case "ping":
			c.handlePing()
		case "help":
			c.printHelp()
		case "quit", "exit", "q":
			fmt.Println("Bye!")
			return nil
		default:
			fmt.Printf("Unexpected command: %s. Type 'help' to get full list of commands.\n\n", command)
		}
	}
	return nil
}

func (c *CLI) printWelcome() {
	fmt.Println("File sevice client CLI")
	fmt.Println("Connected to file server")
	fmt.Println("Type 'help' to get full list of commands.")
}

// handleUpload handles upload command
func (c *CLI) printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  upload <file_path>           - Upload a file to the server")
	fmt.Println("  download <file_id> <path>    - Download a file by ID to specified path")
	fmt.Println("  list                         - List all files on the server")
	fmt.Println("  ping                         - Check server availability")
	fmt.Println("  help                         - Show this help message")
	fmt.Println("  quit/exit/q                  - Exit the client")
	fmt.Println()
}

// handleUpload handles upload command
func (c *CLI) handleUpload(args []string) {
	if len(args) != 1 {
		fmt.Println("Usage: upload <file_path>")
		return
	}

	filePath := args[0]

	// check file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("ERROR: FILE '%s' DOES NOT EXIST\n", filePath)
		return
	}

	fmt.Printf("Uploading file '%s'...\n", filePath)

	start := time.Now()
	fileID, err := c.client.UploadFileFromPath(context.Background(), filePath)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("ERROR UPLOADING FILE: %v\n", err)
		return
	}

	fmt.Printf("File uploaded successfully!\n")
	fmt.Printf("File ID: %s\n", fileID)
	fmt.Printf("Upload time %v\n", duration)
}

// handleDownload handles download command
func (c *CLI) handleDownload(args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: download <file_id> <output_path>")
		return
	}
	fileID := args[0]
	outputPath := args[1]

	fmt.Printf("Downloading file with ID '%s'...\n", fileID)

	start := time.Now()
	err := c.client.DownloadFileToPath(context.Background(), fileID, outputPath)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("ERROR DOWNLOADING FILE: %v\n", err)
		return
	}

	fmt.Printf("File downloaded successfully!")
	fmt.Printf("Downloaded to: %s\n", outputPath)
	fmt.Printf("Download time: %v\n", duration)
}

func (c *CLI) handleList() {
	fmt.Println("Fetching file list...")

	start := time.Now()
	resp, err := c.client.ListFiles(context.Background())
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("ERROR LISTING FILES: %v\n", err)
		return
	}

	if len(resp.Files) == 0 {
		fmt.Println("No files found on the server")
		return
	}

	fmt.Printf("Found %d files(s) (fetched in %v):\n", len(resp.Files), duration)
	fmt.Printf("%-36s %-30s %-20s %-20s\n", "ID", "FILENAME", "CREATED", "UPDATED")
	fmt.Println(strings.Repeat("-", 90))

	for _, file := range resp.Files {
		created := time.Unix(file.CreatedAt, 0).Format("2006-01-02 15:04:05")
		updated := time.Unix(file.UpdatedAt, 0).Format("2006-01-02 15:04:05")

		filename := file.Filename
		if len(filename) > 30 {
			filename = filename[:27] + "..."
		}

		fmt.Printf("%-36s %-30s %-20s %-20s\n", file.FileId, filename, created, updated)
	}
	fmt.Println()
}

func (c *CLI) handlePing() {
	fmt.Println("Ping server")

	start := time.Now()
	err := c.client.Ping(context.Background())
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Failed (%v): %v\n", duration, err)
		return
	}

	fmt.Printf("Pong (%v)\n", duration)
}

func (c *CLI) RunBatch(commands []string) error {
	for _, cmd := range commands {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		switch command {
		case "upload":
			c.handleUpload(args)
		case "download":
			c.handleDownload(args)
		case "list":
			c.handleList()
		case "ping":
			c.handlePing()
		default:
			fmt.Printf("Unexpected command: %s\n", command)
		}
	}
	return nil
}
