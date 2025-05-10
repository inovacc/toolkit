package main

import (
	"fmt"
	"time"

	"github.com/inovacc/utils/v2/time/schedule/cron"
)

func main() {
	c := cron.New()
	_, _ = c.AddFunc("@every 2s", func() { fmt.Println("Every 2 seconds:", time.Now()) })
	c.Start()

	time.Sleep(10 * time.Second)
	c.Stop()
}
