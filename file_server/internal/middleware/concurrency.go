package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// ConcurrencyLimiter - middleware для ограничения количества одновременных запросов
// Предотвращает перегрузку сервера, ограничивая конкурентность операций
type ConcurrencyLimiter struct {
	// uploadSemaphore - семафор для ограничения одновременных операций загрузки/скачивания
	// Буферизованный канал, где каждая отправка = занятие слота
	uploadSemaphore chan struct{}

	// listSemaphore - семафор для ограничения одновременных запросов списка файлов
	// Список файлов менее ресурсоемкий, поэтому лимит выше
	listSemaphore chan struct{}

	// stats - структура для хранения статистики с thread-safe доступом
	stats struct {
		uploadActive int          // Количество активных операций загрузки/скачивания
		listActive   int          // Количество активных запросов списка файлов
		totalUploads int64        // Общее количество выполненных загрузок/скачиваний
		totalLists   int64        // Общее количество выполненных запросов списка
		mutex        sync.RWMutex // Мьютекс для безопасного доступа к статистике
	}
}

// NewConcurrencyLimiter создает новый экземпляр ограничителя конкурентности
// Инициализирует семафоры с предустановленными лимитами:
// - 10 одновременных операций загрузки/скачивания (ресурсоемкие операции)
// - 100 одновременных запросов списка файлов (легкие операции)
func NewConcurrencyLimiter() *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		uploadSemaphore: make(chan struct{}, 10),  // 10 одновременных запросов для загрузки/скачивания файлов
		listSemaphore:   make(chan struct{}, 100), // 100 одновременных запросов для получения списка файлов
	}
}

// UnaryServerInterceptor возвращает gRPC interceptor для ограничения конкурентности
// Анализирует тип запроса и направляет его в соответствующий обработчик
func (cl *ConcurrencyLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Определяем тип операции по имени метода gRPC
		switch {
		// Операции загрузки и скачивания файлов - ресурсоемкие, лимит 10
		case strings.Contains(info.FullMethod, "UploadFile") || strings.Contains(info.FullMethod, "GetFile"):
			return cl.handleUploadDownload(ctx, req, info, handler)

		// Операции получения списка файлов - легкие, лимит 100
		case strings.Contains(info.FullMethod, "ListFiles"):
			return cl.handleList(ctx, req, info, handler)

		// Остальные операции пропускаем без ограничений
		default:
			return handler(ctx, req)
		}
	}
}

// handleUploadDownload обрабатывает запросы загрузки и скачивания файлов
// Ограничивает количество одновременных операций до 10
func (cl *ConcurrencyLimiter) handleUploadDownload(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	select {
	// Пытаемся получить слот в семафоре (неблокирующая операция)
	case cl.uploadSemaphore <- struct{}{}:
		// Увеличиваем счетчик активных операций
		cl.updateUploadStats(1)

		// defer гарантирует освобождение слота и обновление статистики при выходе из функции
		defer func() {
			<-cl.uploadSemaphore     // Освобождаем слот
			cl.updateUploadStats(-1) // Уменьшаем счетчик активных операций
		}()

		// Искусственная задержка для тестирования ограничений конкурентности
		time.Sleep(500 * time.Millisecond)

		// Выполняем оригинальный обработчик запроса
		return handler(ctx, req)

	// Проверяем, не был ли отменен контекст запроса
	case <-ctx.Done():
		return nil, ctx.Err()

	// Если семафор заполнен (все 10 слотов заняты), возвращаем ошибку
	default:
		return nil, fmt.Errorf("TOO MANY CONC UPLOAD/DOWNLOAD REQUESTS, MAX 10")
	}
}

// handleList обрабатывает запросы получения списка файлов
// Ограничивает количество одновременных операций до 100
func (cl *ConcurrencyLimiter) handleList(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	select {
	// Пытаемся получить слот в семафоре для операций со списком (неблокирующая операция)
	case cl.listSemaphore <- struct{}{}:
		// Увеличиваем счетчик активных операций со списком
		cl.updateListStats(1)

		// defer гарантирует освобождение слота и обновление статистики при выходе из функции
		defer func() {
			<-cl.listSemaphore     // Освобождаем слот
			cl.updateListStats(-1) // Уменьшаем счетчик активных операций
		}()

		// Искусственная задержка для тестирования ограничений конкурентности (больше чем для загрузки)
		time.Sleep(1500 * time.Millisecond)

		// Выполняем оригинальный обработчик запроса
		return handler(ctx, req)

	// Проверяем, не был ли отменен контекст запроса
	case <-ctx.Done():
		return nil, ctx.Err()

	// Если семафор заполнен (все 100 слотов заняты), возвращаем ошибку
	default:
		return nil, fmt.Errorf("TOO MANY CONC LIST REQUESTS, MAX 100")
	}
}

// updateUploadStats thread-safe обновление статистики операций загрузки/скачивания
// delta: +1 при начале операции, -1 при завершении
func (cl *ConcurrencyLimiter) updateUploadStats(delta int) {
	// Блокируем мьютекс для эксклюзивного доступа к статистике
	cl.stats.mutex.Lock()
	defer cl.stats.mutex.Unlock() // Гарантированно разблокируем при выходе из функции

	// Обновляем количество активных операций
	cl.stats.uploadActive += delta

	// Если операция начинается (delta > 0), увеличиваем общий счетчик
	if delta > 0 {
		cl.stats.totalUploads++
	}
}

// updateListStats thread-safe обновление статистики операций со списком файлов
// delta: +1 при начале операции, -1 при завершении
func (cl *ConcurrencyLimiter) updateListStats(delta int) {
	// Блокируем мьютекс для эксклюзивного доступа к статистике
	cl.stats.mutex.Lock()
	defer cl.stats.mutex.Unlock() // Гарантированно разблокируем при выходе из функции

	// Обновляем количество активных операций со списком
	cl.stats.listActive += delta

	// Если операция начинается (delta > 0), увеличиваем общий счетчик
	if delta > 0 {
		cl.stats.totalLists++
	}
}

// GetStats возвращает текущую статистику операций
// Использует read-lock для безопасного чтения без блокировки записи
func (cl *ConcurrencyLimiter) GetStats() (uploadActive, listActive int, totalUploads, totalLists int64) {
	// Блокируем read-lock для безопасного чтения статистики
	cl.stats.mutex.RLock()
	defer cl.stats.mutex.RUnlock() // Гарантированно разблокируем при выходе из функции

	// Возвращаем все счетчики статистики
	return cl.stats.uploadActive, cl.stats.listActive, cl.stats.totalUploads, cl.stats.totalLists
}

// GetStatsString форматирует статистику в читаемую строку для логирования/мониторинга
// Показывает текущее использование лимитов и общую статистику
func (cl *ConcurrencyLimiter) GetStatsString() string {
	// Получаем актуальную статистику
	uploadActive, listActive, totalUploads, totalLists := cl.GetStats()

	// Форматируем строку с информацией о текущем использовании и общих счетчиках
	return fmt.Sprintf("Upload/Download: %d/10 active, %d total, | List: %d/100 active, %d total",
		uploadActive, totalUploads, listActive, totalLists)
}
