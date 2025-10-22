// main.go - точка входа для файлового gRPC сервера
// Настраивает и запускает сервер с ограничениями конкурентности и graceful shutdown
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

// serviceName - имя сервиса для логирования
const serviceName = "file-service"

// main - основная функция приложения
// Инициализирует все компоненты сервера и запускает gRPC сервер
func main() {
	// Парсинг аргументов командной строки
	var (
		port        = flag.Int("port", 8080, "Server port")                               // Порт для gRPC сервера
		storagePath = flag.String("storage", "./storage/files", "Storage Directory Path") // Путь к директории хранения файлов
		showStats   = flag.Bool("stats", false, "Show concurrency statistics")            // Флаг для отображения статистики конкурентности
	)
	flag.Parse()

	// Логирование информации о запуске сервера
	log.Printf("Start %s on port %d", serviceName, *port)
	log.Printf("Storage Directory: %s", *storagePath)
	log.Printf("Concurrency limits: Upload/Download=10, List=100")

	// Создание репозитория для работы с файлами
	// Репозиторий отвечает за сохранение, загрузку и управление файлами на диске
	repo, err := filerepo.NewRepo(*storagePath)
	if err != nil {
		log.Fatalf("FAILED TO CREATE REPOSITORY: %v", err)
	}

	// Создание контроллера для обработки бизнес-логики
	// Контроллер координирует работу между gRPC обработчиком и репозиторием
	ctrl := filectrl.NewController(repo)

	// Создание gRPC обработчика
	// Обработчик преобразует gRPC запросы в вызовы контроллера
	grpcHandler := filegrpc.NewGrpc(ctrl)

	// Создание middleware для ограничения конкурентности
	// Middleware предотвращает перегрузку сервера, ограничивая количество одновременных запросов
	concurrencyLimiter := middleware.NewConcurrencyLimiter()

	// Настройка TCP listener для gRPC сервера
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("FAILED TO LISTEN: %v", err)
	}

	// Создание gRPC сервера с настройками
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(concurrencyLimiter.UnaryServerInterceptor()), // Подключение middleware для ограничения конкурентности
		grpc.MaxConcurrentStreams(200),                                     // Максимум 200 одновременных потоков
	)

	// Регистрация сервиса и включение reflection для отладки
	gen.RegisterFileServiceServer(srv, grpcHandler) // Регистрация файлового сервиса
	reflection.Register(srv)                        // Включение gRPC reflection для интроспекции API

	// Запуск горутины для мониторинга статистики конкурентности (если включен флаг --stats)
	if *showStats {
		go func() {
			ticker := time.NewTicker(5 * time.Second) // Таймер для периодического вывода статистики
			defer ticker.Stop()

			// Бесконечный цикл для периодического вывода статистики
			for range ticker.C {
				stats := concurrencyLimiter.GetStatsString()
				log.Printf("Concurrency stats: %s", stats)
			}
		}()
	}

	// Настройка graceful shutdown - обработка сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // Перехват сигналов SIGINT (Ctrl+C) и SIGTERM

	// Горутина для обработки сигналов завершения
	go func() {
		<-sigChan // Ожидание сигнала завершения
		log.Printf("Recieved interrupt signal. Shutting down..")

		// Настройка graceful shutdown с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Горутина для принудительного завершения при превышении таймаута
		go func() {
			<-ctx.Done()
			log.Println("Shutdown timeout exceeded. Exiting")
			os.Exit(1)
		}()

		// Graceful остановка gRPC сервера
		srv.GracefulStop()
		log.Println("Server stopped gracefully")
		os.Exit(0)
	}()

	// Запуск gRPC сервера
	log.Printf("Server is ready to accept connections on localhost:%d", *port)
	log.Printf("Use Ctrl+C to stop the server")

	// Блокирующий вызов для запуска сервера
	// Сервер будет обрабатывать входящие соединения до получения сигнала завершения
	if err := srv.Serve(listener); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
