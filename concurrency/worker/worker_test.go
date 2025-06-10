package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBaseWorkerProcessesInputs(t *testing.T) {
	var (
		mu       sync.Mutex
		received []string
	)

	logger := StdLogger{}

	workload := func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-input:
				if !ok {
					return
				}
				if str, ok := msg.(string); ok {
					mu.Lock()
					received = append(received, str)
					mu.Unlock()
				}
			}
		}
	}

	worker := NewBaseWorker("test-worker", logger, workload, 100)
	ctx := context.Background()
	worker.Start(ctx)

	inputs := []string{"a", "b", "c"}
	for _, val := range inputs {
		if err := worker.Send(val); err != nil {
			return
		}
	}

	time.Sleep(200 * time.Millisecond)
	worker.Stop()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, inputs, received)
}

func TestWorkerStopsOnContextCancel(t *testing.T) {
	done := make(chan struct{})
	logger := StdLogger{}

	workload := func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			case <-input:
				// simulate work
			}
		}
	}

	worker := NewBaseWorker("cancel-test", logger, workload, 100)

	ctx, cancel := context.WithCancel(context.Background())
	worker.Start(ctx)

	if err := worker.Send("will be ignored"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("worker did not stop after context cancel")
	}
}

func TestWorkerConcurrentInput(t *testing.T) {
	var (
		mu       sync.Mutex
		received = make(map[int]bool)
	)
	logger := StdLogger{}

	workload := func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-input:
				if v, ok := msg.(int); ok {
					mu.Lock()
					received[v] = true
					mu.Unlock()
				}
			}
		}
	}

	worker := NewBaseWorker("concurrent-test", logger, workload, 100)
	ctx := context.Background()
	worker.Start(ctx)

	const total = 100
	var wg sync.WaitGroup
	wg.Add(total)

	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()
			if err := worker.Send(i); err != nil {
				return
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)
	worker.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, total, len(received))
}

func TestSendAfterStopIsSafe(t *testing.T) {
	logger := StdLogger{}

	workload := func(ctx context.Context, log Logger, input <-chan any) {
		for range input {
		}
	}

	worker := NewBaseWorker("post-stop", logger, workload, 100)
	ctx := context.Background()
	worker.Start(ctx)
	worker.Stop()

	assert.NotPanics(t, func() {
		if err := worker.Send("ignored"); err != nil {
			return
		}
	})
}

func TestWorkerSendChannelFull(t *testing.T) {
	logger := StdLogger{}
	capacity := 5
	worker := NewBaseWorker("overflow-test", logger, func(ctx context.Context, log Logger, input <-chan any) {
		<-ctx.Done() // no lee del canal
	}, capacity)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx)

	for i := 0; i < capacity; i++ {
		assert.NoError(t, worker.Send(i))
	}

	err := worker.Send("overflow")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input channel is full")

	worker.Stop()
}

func TestWorkerBackpressureBehavior(t *testing.T) {
	logger := StdLogger{}
	bufferSize := 10
	totalSends := 100

	var received []int
	var mu sync.Mutex

	worker := NewBaseWorker("backpressure", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-input:
				if !ok {
					return
				}
				time.Sleep(2 * time.Millisecond) // simulate slow consumer
				if i, ok := val.(int); ok {
					mu.Lock()
					received = append(received, i)
					mu.Unlock()
				}
			}
		}
	}, bufferSize)

	ctx := context.Background()
	worker.Start(ctx)

	drops := 0
	for i := 0; i < totalSends; i++ {
		if err := worker.Send(i); err != nil {
			drops++
		}
	}

	worker.Stop()

	t.Logf("Dropped: %d / %d", drops, totalSends)
	mu.Lock()
	defer mu.Unlock()
	t.Logf("Processed: %d", len(received))
	assert.GreaterOrEqual(t, drops, 85)
}

func TestWorkerMultipleStopStart(t *testing.T) {
	logger := StdLogger{}
	started := make(chan struct{})

	workload := func(ctx context.Context, log Logger, input <-chan any) {
		close(started)
		<-ctx.Done()
	}

	worker := NewBaseWorker("restarts", logger, workload, 10)

	ctx := context.Background()
	worker.Start(ctx)
	worker.Start(ctx) // should be ignored
	worker.Stop()
	worker.Stop() // should be ignored

	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("worker workload did not run")
	}
}
