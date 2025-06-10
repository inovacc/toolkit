package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Logger defines an interface for logging.
type Logger interface {
	Info(format string, a ...any)
	Error(format string, a ...any)
}

// StdLogger is a default implementation using Go's log package.
type StdLogger struct{}

// Info logs an informational message using formatting.
func (l StdLogger) Info(format string, a ...any) {
	log.Printf("[INFO] %s", formatWithArgs(format, a...))
}

// Error logs an error message using formatting.
func (l StdLogger) Error(format string, a ...any) {
	log.Printf("[ERROR] %s", formatWithArgs(format, a...))
}

// formatWithArgs handles the formatting inline.
func formatWithArgs(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}

// Workload defines the task a worker should perform.
type Workload func(ctx context.Context, log Logger, input <-chan any)

// Worker defines the worker interface.
type Worker interface {
	Start(ctx context.Context)
	Stop()
	Name() string
}

// BaseWorker runs a workload in a cancellable goroutine.
type BaseWorker struct {
	name     string
	logger   Logger
	workload Workload
	cancel   context.CancelFunc
	running  sync.Once
	stopped  sync.Once
	inputCh  chan any
	mu       sync.RWMutex
	closed   bool
}

// NewBaseWorker creates a new worker with custom logger and workload.
func NewBaseWorker(name string, logger Logger, workload Workload, bufferSize int) *BaseWorker {
	return &BaseWorker{
		name:     name,
		logger:   logger,
		workload: workload,
		inputCh:  make(chan any, bufferSize),
	}
}

func (w *BaseWorker) Start(ctx context.Context) {
	w.running.Do(func() {
		var runCtx context.Context
		runCtx, w.cancel = context.WithCancel(ctx)

		go func() {
			w.logger.Info("Worker %s started", w.name)
			defer w.logger.Info("Worker %s stopped", w.name)
			w.workload(runCtx, w.logger, w.inputCh)
		}()
	})
}

func (w *BaseWorker) Stop() {
	w.stopped.Do(func() {
		w.mu.Lock()
		w.closed = true
		w.mu.Unlock()

		if w.cancel != nil {
			w.cancel()
		}
		close(w.inputCh)
	})
}

func (w *BaseWorker) Name() string {
	return w.name
}

func (w *BaseWorker) Send(input any) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return fmt.Errorf("worker %s input channel is closed", w.name)
	}

	select {
	case w.inputCh <- input:
		return nil
	default:
		return fmt.Errorf("worker %s input channel is full", w.name)
	}
}
