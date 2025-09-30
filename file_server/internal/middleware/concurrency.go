package middleware

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

type ConcurrencyLimiter struct {
	uploadSemaphore chan struct{}
	listSemaphore   chan struct{}

	stats struct {
		uploadActive int
		listActive   int
		totalUploads int64
		totalLists   int64
		mutex        sync.RWMutex
	}
}

func NewConcurrencyLimiter() *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		uploadSemaphore: make(chan struct{}, 10),  // 10 conc req for downloading/uploading files
		listSemaphore:   make(chan struct{}, 100), // 100 conc req for searching listfiles
	}
}

func (cl *ConcurrencyLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		switch {
		case strings.Contains(info.FullMethod, "UploadFile") || strings.Contains(info.FullMethod, "GetFile"):
			return cl.handleUploadDownload(ctx, req, info, handler)

		case strings.Contains(info.FullMethod, "ListFiles"):
			return cl.handleList(ctx, req, info, handler)

		default:
			return handler(ctx, req)
		}
	}
}

func (cl *ConcurrencyLimiter) handleUploadDownload(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	select {
	case cl.uploadSemaphore <- struct{}{}:
		cl.updateUploadStats(1)
		defer func() {
			<-cl.uploadSemaphore
			cl.updateUploadStats(-1)
		}()

		return handler(ctx, req)

	case <-ctx.Done():
		return nil, ctx.Err()

	default:
		return nil, fmt.Errorf("Too many conc upload/download requests, max 10")
	}
}

func (cl *ConcurrencyLimiter) handleList(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	select {
	case cl.uploadSemaphore <- struct{}{}:
		cl.updateListStats(1)
		defer func() {
			<-cl.uploadSemaphore
			cl.updateListStats(-1)
		}()

		return handler(ctx, req)

	case <-ctx.Done():
		return nil, ctx.Err()

	default:
		return nil, fmt.Errorf("Too many conc list requests, max 10")
	}
}

func (cl *ConcurrencyLimiter) updateUploadStats(delta int) {
	cl.stats.mutex.Lock()
	defer cl.stats.mutex.Unlock()

	cl.stats.uploadActive += delta
	if delta > 0 {
		cl.stats.totalUploads++
	}
}

func (cl *ConcurrencyLimiter) updateListStats(delta int) {
	cl.stats.mutex.Lock()
	defer cl.stats.mutex.Unlock()

	cl.stats.listActive += delta
	if delta > 0 {
		cl.stats.totalLists++
	}
}

func (cl *ConcurrencyLimiter) GetStats() (uploadActive, listActive int, totalUploads, totalLists int64) {
	cl.stats.mutex.RLock()
	defer cl.stats.mutex.RUnlock()

	return cl.stats.uploadActive, cl.stats.listActive, cl.stats.totalUploads, cl.stats.totalLists
}

func (cl *ConcurrencyLimiter) GetStatsString() string {
	uploadActive, listActive, totalUploads, totalLists := cl.GetStats()

	return fmt.Sprintf("Upload/Download: %d/10 active, %d total, | List: %d/100 active, %d total",
		uploadActive, totalUploads, listActive, totalLists)
}
