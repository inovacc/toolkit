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
