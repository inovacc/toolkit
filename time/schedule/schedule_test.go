package schedule

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewCronScheduler(t *testing.T) {
	sc, err := NewCronScheduler(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	var (
		mu      sync.Mutex
		message = "waiting"
	)

	done := func() {
		mu.Lock()
		message = "done"
		mu.Unlock()
	}

	id, err := sc.AddFunc("@minute", done)
	if err != nil {
		t.Fatal(err)
	}

	<-time.After(time.Second * 61)

	mu.Lock()
	defer mu.Unlock()
	fmt.Println(id, message)

	if message != "done" {
		t.Fatalf("expecting message 'done', got '%s'", message)
	}
}

func TestNewCronScheduler_EverySecond(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sc, err := NewCronScheduler(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var (
		wg      sync.WaitGroup
		message = "waiting"
		mu      sync.Mutex
	)

	wg.Add(1)
	job := func() {
		mu.Lock()
		message = "done"
		mu.Unlock()
		wg.Done()
	}

	if _, err = sc.AddFunc("@every 1s", job); err != nil {
		t.Fatal(err)
	}

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		mu.Lock()
		defer mu.Unlock()
		if message != "done" {
			t.Fatalf("expected message 'done', got '%s'", message)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for cron job to run")
	}
}

func TestCronScheduler_WeekdayMapping(t *testing.T) {
	specs := map[string]string{
		Weekday:   "0 0 0 * * 1-5",
		Monday:    "0 0 0 * * 1",
		Tuesday:   "0 0 0 * * 2",
		Wednesday: "0 0 0 * * 3",
		Thursday:  "0 0 0 * * 4",
		Friday:    "0 0 0 * * 5",
		Saturday:  "0 0 0 * * 6",
		Sunday:    "0 0 0 * * 0",
		Minute:    "0 * * * * *",
	}

	c := &Cron{}
	for input, expected := range specs {
		actual := c.fixWeekday(input)
		if actual != expected {
			t.Errorf("Expected %q for %q, got %q", expected, input, actual)
		}
	}
}

func TestNewCronScheduler_ContextNil(t *testing.T) {
	if _, err := NewCronScheduler(nil); err == nil {
		t.Fatal("expected error for nil context, got nil")
	}
}

func TestCronScheduler_MultipleJobs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sc, err := NewCronScheduler(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	results := make([]string, 0, 2)
	appendResult := func(msg string) func() {
		return func() {
			mu.Lock()
			defer mu.Unlock()
			results = append(results, msg)
		}
	}

	if _, err = sc.AddFunc("@every 1s", appendResult("job1")); err != nil {
		t.Fatal(err)
	}

	if _, err = sc.AddFunc("@every 1s", appendResult("job2")); err != nil {
		t.Fatal(err)
	}

	<-ctx.Done()

	mu.Lock()
	defer mu.Unlock()
	if len(results) == 0 {
		t.Fatal("expected at least one job to run")
	}
}

func TestCronScheduler_EveryDurationValid(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sc, err := NewCronScheduler(ctx)
	if err != nil {
		t.Fatal(err)
	}

	called := make(chan struct{})
	_, err = sc.AddFunc("@every 1s", func() {
		called <- struct{}{}
	})
	if err != nil {
		t.Fatalf("expected valid schedule, got error: %v", err)
	}

	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("expected job to execute within 3s")
	}
}

func TestCronScheduler_InvalidSpec(t *testing.T) {
	sc, err := NewCronScheduler(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	_, err = sc.AddFunc("@nonsense", func() {})
	if err == nil {
		t.Fatal("expected error for invalid cron spec")
	}
}

func TestCronScheduler_DuplicateJobs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sc, err := NewCronScheduler(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	count := 0
	job := func() {
		mu.Lock()
		count++
		mu.Unlock()
	}

	for i := 0; i < 3; i++ {
		if _, err := sc.AddFunc("@every 1s", job); err != nil {
			t.Fatalf("could not schedule job %d: %v", i+1, err)
		}
	}

	<-ctx.Done()
	mu.Lock()
	defer mu.Unlock()

	if count < 3 {
		t.Errorf("expected job to run multiple times, got count: %d", count)
	}
}

func TestCronScheduler_ConcurrentAddFunc(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	sc, err := NewCronScheduler(ctx)
	if err != nil {
		t.Fatal(err)
	}

	const numJobs = 100
	var wg sync.WaitGroup
	wg.Add(numJobs)

	for i := 0; i < numJobs; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := sc.AddFunc("@every 1s", func() {
				// noop
			})
			if err != nil {
				t.Errorf("failed to add job %d: %v", i, err)
			}
		}(i)
	}

	wg.Wait()
}
