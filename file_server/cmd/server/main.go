package main

import (
	"context"
	"file_server/gen"
	filectrl "file_server/internal/controller/file"
	filegrpc "file_server/internal/handler/grpc"
	"file_server/internal/middleware"
	filerepo "file_server/internal/repository/file"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const serviceName = "file-service"

func main() {
	// parse arguments from cli
	var (
		port        = flag.Int("port", 8080, "Server port")
		storagePath = flag.String("storage", "./storage/files", "Storage Directory Path")
		showStats   = flag.Bool("stats", false, "Show concurrency statistics")
	)
	flag.Parse()

	// logging info
	log.Printf("Start %s on port %d", serviceName, *port)
	log.Printf("Storage Directory: %s", *storagePath)
	log.Printf("Concurrency limits: Upload/Download=10, List=100")

	// creating repo for files
	repo, err := filerepo.NewRepo(*storagePath)
	if err != nil {
		log.Fatalf("FAILED TO CREATE REPOSITORY: %v", err)
	}

	// creating controller
	ctrl := filectrl.NewController(repo)

	// creating grpc handler
	grpcHandler := filegrpc.NewGrpc(ctrl)

	// creating middleware for concurrency limitations
	concurrencyLimiter := middleware.NewConcurrencyLimiter()

	// setting grpc Server
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("FAILED TO LISTEN: %v", err)
	}

	// creating grpc server w/ concurrencyLimiter
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(concurrencyLimiter.UnaryServerInterceptor()),
	)

	// registering service
	gen.RegisterFileServiceServer(srv, grpcHandler)
	reflection.Register(srv)

	// starting goroutine for stat monitoring
	if *showStats {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				stats := concurrencyLimiter.GetStatsString()
				log.Printf("Concurrency stats: %s", stats)
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Recieved interrupt signal. Shutting down..")

		// shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		go func() {
			<-ctx.Done()
			log.Println("Shutdown timeout exceeded. Exiting")
			os.Exit(1)
		}()

		srv.GracefulStop()
		log.Println("Server stopped gracefully")
		os.Exit(0)
	}()

	// starting server
	log.Printf("Server is ready to accept connections on localhost:%d", *port)
	log.Printf("Use Ctrl+C to stop the server")

	if err := srv.Serve(listener); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
