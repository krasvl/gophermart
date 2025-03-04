package main

import (
	"log"

	"github.com/krasvl/market/internal/scheduler"
)

func main() {
	databaseDefault := ""
	accrualAddrDefault := ""

	scheduler, err := scheduler.GetConfiguredScheduler(databaseDefault, accrualAddrDefault)
	if err != nil {
		log.Fatalf("Scheduler configure error: %v", err)
	}

	scheduler.Start()
}
