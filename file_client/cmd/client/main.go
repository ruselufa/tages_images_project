package main

import (
	"context"
	"file_client/internal/client/file"
	"file_client/internal/ui/cli"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		serverAddress = flag.String("server", "localhost:8080", "File service server address")
		batchMode     = flag.Bool("batch", false, "Run in batch mode")
		timeout       = flag.Duration("timeout", 30*time.Second, "Connection timeout")
	)

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	fmt.Printf("Connection to file service at %s...\n", *serverAddress)
	fileClient, err := file.NewClient(*serverAddress)
	if err != nil {
		log.Fatalf("FAILED TO CREATE CLIENT: %v", err)
	}
	defer fileClient.Close()

	// Checking Connection
	if err := fileClient.Ping(ctx); err != nil {
		log.Fatalf("FAILED TO CONNECT TO SERVER: %v", err)
	}

	fmt.Println("Connected to file service succesfully")

	cli := cli.New(fileClient)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nRecieved interrupt signal. Shutting down...")
		os.Exit(0)
	}()

	// Starting CLIENT
	if *batchMode {
		commands := flag.Args()
		if len(commands) == 0 {
			fmt.Println("No commands been given for batch mode")
			return
		}

		if err := cli.RunBatch(commands); err != nil {
			log.Fatalf("BATCH MODE ERROR: %v", err)
		}
	} else {
		if err := cli.Run(); err != nil {
			log.Fatalf("CLI ERROR: %v", err)
		}
	}
}
