package schedule

import (
	"context"
	"testing"
)

func BenchmarkCronScheduler_AddFunc(b *testing.B) {
	sc, err := NewCronScheduler(context.Background())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sc.AddFunc("@every 1m", func() {}); err != nil {
			b.Fatalf("error adding job: %v", err)
		}
	}
}
