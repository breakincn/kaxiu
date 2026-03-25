package main

import (
	"encoding/json"
	"flag"
	"kabao/config"
	"kabao/scheduler"
	"log"
	"os"
	"time"
)

func main() {
	var staleMinutes = flag.Int("stale-minutes", 3, "mark scheduler unhealthy after this many minutes without a tick")
	flag.Parse()

	config.LoadEnv()
	config.InitDB()

	report, err := scheduler.GetSchedulerHealthReport(config.DB, time.Now(), *staleMinutes)
	if err != nil {
		log.Fatal("读取调度器健康状态失败:", err)
	}

	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatal("序列化调度器健康状态失败:", err)
	}
	log.Println(string(out))
	if report.OverallStatus != "healthy" {
		os.Exit(2)
	}
}
