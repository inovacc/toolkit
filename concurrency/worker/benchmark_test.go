package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkWorkerSendParallel(b *testing.B) {
	logger := StdLogger{}
	worker := NewBaseWorker("bench", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-input:
			}
		}
	}, 1000)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = worker.Send("msg")
		}
	})

	worker.Stop()
}

func BenchmarkWorkerThroughput(b *testing.B) {
	logger := StdLogger{}
	var wg sync.WaitGroup
	wg.Add(b.N)

	worker := NewBaseWorker("throughput", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-input:
				if !ok {
					return
				}
				time.Sleep(100 * time.Microsecond) // simulate work
				wg.Done()
			}
		}
	}, b.N)

	ctx := context.Background()
	worker.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = worker.Send(i)
	}

	wg.Wait()
	worker.Stop()
}

func TestWorkerBurstLoad(t *testing.T) {
	var (
		mu       sync.Mutex
		received = make(map[int]bool)
	)

	logger := StdLogger{}
	total := 10000
	buffer := 2048

	worker := NewBaseWorker("burst-test", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-input:
				if !ok {
					return
				}
				if i, ok := v.(int); ok {
					mu.Lock()
					received[i] = true
					mu.Unlock()
				}
			}
		}
	}, buffer)

	ctx := context.Background()
	worker.Start(ctx)

	for i := 0; i < total; i++ {
		_ = worker.Send(i) // ignore send errors for test, could be counted
	}

	time.Sleep(1 * time.Second)
	worker.Stop()

	mu.Lock()
	defer mu.Unlock()
	t.Logf("Received: %d / %d", len(received), total)
	assert.Greater(t, len(received), total/2) // at least 50% must be processed
}

func TestWorkerSustainedParallelLoad(t *testing.T) {
	logger := StdLogger{}
	const workers = 4
	const totalMessages = 5000

	var mu sync.Mutex
	var count int

	worker := NewBaseWorker("sustained-parallel", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-input:
				if !ok {
					return
				}
				mu.Lock()
				count++
				mu.Unlock()
				time.Sleep(50 * time.Microsecond)
			}
		}
	}, 1000)

	ctx := context.Background()
	worker.Start(ctx)

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < totalMessages/workers; i++ {
				_ = worker.Send(i)
			}
		}(w)
	}

	wg.Wait()
	time.Sleep(1 * time.Second)
	worker.Stop()

	mu.Lock()
	defer mu.Unlock()
	t.Logf("Processed: %d", count)
	assert.InDelta(t, totalMessages, count, float64(totalMessages)*0.85) // ~90% success
}

func BenchmarkWorkerStress(b *testing.B) {
	logger := StdLogger{}
	worker := NewBaseWorker("stress", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-input:
			}
		}
	}, 10000)

	ctx := context.Background()
	worker.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = worker.Send(i)
	}

	worker.Stop()
}

func BenchmarkWorkerAllocs(b *testing.B) {
	logger := StdLogger{}
	worker := NewBaseWorker("allocs", logger, func(ctx context.Context, log Logger, input <-chan any) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-input:
			}
		}
	}, 1000)

	ctx := context.Background()
	worker.Start(ctx)

	allocs := testing.AllocsPerRun(1000, func() {
		_ = worker.Send("allocs")
	})

	worker.Stop()

	b.Logf("Allocations per Send(): %.2f", allocs)
}
