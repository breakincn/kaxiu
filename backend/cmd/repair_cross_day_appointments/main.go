package main

import (
	"encoding/json"
	"flag"
	"kabao/config"
	"kabao/handlers"
	"log"
	"time"
)

func main() {
	var (
		limit  = flag.Int("limit", 200, "maximum appointments to scan")
		dryRun = flag.Bool("dry-run", true, "only list candidates without applying repairs")
	)
	flag.Parse()

	config.LoadEnv()
	config.InitDB()

	now := time.Now().In(time.Local)
	results, err := handlers.RepairCrossDayUnfinishedAppointments(config.DB, now, *limit, *dryRun, "系统批量巡检自动结案：跨日未开始服务")
	if err != nil {
		log.Fatal("批量巡检失败:", err)
	}

	out, err := json.MarshalIndent(map[string]interface{}{
		"dry_run": *dryRun,
		"count":   len(results),
		"data":    results,
	}, "", "  ")
	if err != nil {
		log.Fatal("输出结果失败:", err)
	}
	log.Println(string(out))
}
