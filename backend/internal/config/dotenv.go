package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var dotenvOnce sync.Once

func loadDotEnv() {
	dotenvOnce.Do(func() {
		path := strings.TrimSpace(os.Getenv("SCOREHUB_ENV_FILE"))
		if path != "" {
			_ = loadDotEnvFile(path)
			return
		}

		if p := findDotEnv(); p != "" {
			_ = loadDotEnvFile(p)
		}
	})
}

func findDotEnv() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := wd
	for i := 0; i < 10; i++ {
		p := filepath.Join(dir, ".env")
		if _, err := os.Stat(p); err == nil {
			return p
		}

		p = filepath.Join(dir, "backend", ".env")
		if _, err := os.Stat(p); err == nil {
			return p
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func loadDotEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue // 不覆盖外部环境变量
		}
		val := strings.TrimSpace(v)
		val = strings.TrimSuffix(val, "\r")
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		_ = os.Setenv(key, val)
	}
	return scanner.Err()
}
