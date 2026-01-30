package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func LoadEnv() {
	candidates := []string{".env", "../.env", "../../.env"}
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	for _, rel := range candidates {
		p := filepath.Clean(filepath.Join(wd, rel))
		if _, err := os.Stat(p); err == nil {
			loadEnvFile(p)
			return
		}
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		val = strings.Trim(val, "\"'")
		_ = os.Setenv(key, val)
	}
}
