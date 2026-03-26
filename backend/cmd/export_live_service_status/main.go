package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"kabao/config"
	"kabao/handlers"
	"kabao/models"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	var merchantIDsRaw = flag.String("merchant-ids", "", "comma-separated merchant IDs")
	var outPath = flag.String("out", "", "markdown output path")
	flag.Parse()

	if strings.TrimSpace(*merchantIDsRaw) == "" {
		log.Fatal("merchant-ids 不能为空，例如 --merchant-ids=12,18,26")
	}

	merchantIDs, err := parseMerchantIDs(*merchantIDsRaw)
	if err != nil {
		log.Fatal(err)
	}

	config.LoadEnv()
	config.InitDB()

	now := time.Now()
	report, err := buildReport(merchantIDs, now)
	if err != nil {
		log.Fatal(err)
	}

	targetPath := strings.TrimSpace(*outPath)
	if targetPath == "" {
		targetPath = filepath.Join("..", "docs", "plans", fmt.Sprintf("门店实时服务状态抽样快照_%s.md", now.Format("20060102_150405")))
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte(report), 0o644); err != nil {
		log.Fatal(err)
	}

	log.Printf("导出完成: %s\n", targetPath)
}

func parseMerchantIDs(raw string) ([]uint, error) {
	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	seen := map[uint]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil || n == 0 {
			return nil, fmt.Errorf("无效 merchant id: %s", part)
		}
		id := uint(n)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("merchant-ids 不能为空")
	}
	return out, nil
}

func buildReport(merchantIDs []uint, now time.Time) (string, error) {
	var b strings.Builder
	b.WriteString("# 门店实时服务状态抽样快照\n\n")
	b.WriteString("生成时间：")
	b.WriteString(now.Format("2006-01-02 15:04:05"))
	b.WriteString("\n\n")

	for _, merchantID := range merchantIDs {
		var merchant models.Merchant
		if err := config.DB.Select("id", "name", "type").First(&merchant, merchantID).Error; err != nil {
			return "", err
		}

		snapshot, err := handlers.BuildMerchantLiveServiceStatusSnapshot(merchantID, now)
		if err != nil {
			return "", err
		}
		payload, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			return "", err
		}

		b.WriteString("## ")
		b.WriteString(merchant.Name)
		b.WriteString("\n\n")
		b.WriteString("- 商户ID：")
		b.WriteString(strconv.FormatUint(uint64(merchant.ID), 10))
		b.WriteString("\n")
		b.WriteString("- 门店类型：")
		b.WriteString(strings.TrimSpace(merchant.Type))
		b.WriteString("\n")
		b.WriteString("- 快照时间：")
		b.WriteString(now.Format("2006-01-02 15:04:05"))
		b.WriteString("\n\n")
		b.WriteString("```json\n")
		b.Write(payload)
		b.WriteString("\n```\n\n")
	}

	return b.String(), nil
}
