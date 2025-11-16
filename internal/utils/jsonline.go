package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type dailyLogger struct {
	file          *os.File
	currentDate   string
	mu            sync.Mutex
	retentionDays int
	lastCleanup   time.Time
	dir           string
	category      string
}

var (
	loggers  = make(map[string]*dailyLogger)
	globalMu sync.Mutex
)

// GetLogger 依 category 取得 logger（例如 cpu、disk、ram）
func GetLogger(dir, category string, retentionDays int) *dailyLogger {
	key := filepath.Join(dir, category)

	globalMu.Lock()
	defer globalMu.Unlock()

	if lg, ok := loggers[key]; ok {
		return lg
	}

	lg := &dailyLogger{
		dir:           dir,
		category:      category,
		retentionDays: retentionDays,
	}
	loggers[key] = lg
	return lg
}

// Write one JSON line into today's log file
func (lg *dailyLogger) Write(data any) error {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	today := time.Now().Format("2006-01-02")

	// 如果日期變了 → 關閉舊檔 → 開新檔
	if lg.file == nil || lg.currentDate != today {
		if err := lg.rotate(today); err != nil {
			return err
		}
	}

	var b []byte
	var err error

	switch v := data.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		b, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	if _, err := lg.file.Write(append(b, '\n')); err != nil {
		return err
	}

	// 一天只清一次舊檔
	if time.Since(lg.lastCleanup) > 24*time.Hour {
		lg.cleanup()
	}

	return nil
}

// rotate 切換到新的一天的檔案
func (lg *dailyLogger) rotate(today string) error {
	// 關閉舊檔
	if lg.file != nil {
		lg.file.Close()
	}

	if err := os.MkdirAll(lg.dir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-%s.jsonl", lg.category, today)
	fullPath := filepath.Join(lg.dir, filename)

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	lg.file = f
	lg.currentDate = today

	return nil
}

// cleanup 刪除超過 N 天的日誌（一日僅執行一次）
func (lg *dailyLogger) cleanup() {
	lg.lastCleanup = time.Now()

	files, err := os.ReadDir(lg.dir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -lg.retentionDays)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()

		// 檔名格式 category-YYYY-MM-DD.jsonl
		var datePart string
		_, err := fmt.Sscanf(name, lg.category+"-%s.jsonl", &datePart)
		if err != nil {
			continue
		}

		// 移除多餘字元
		if len(datePart) > 10 {
			datePart = datePart[:10]
		}

		d, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}

		if d.Before(cutoff) {
			os.Remove(filepath.Join(lg.dir, name))
		}
	}
}
