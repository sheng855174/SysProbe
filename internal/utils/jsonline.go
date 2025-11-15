package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func WriteJSONLine(dir, filename string, data any) error {
	// 目錄不存在就建立
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 組路徑
	fullPath := filepath.Join(dir, filename)

	// 打開（或建立）要 append 的檔案
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	var b []byte

	switch v := data.(type) {
	case string:
		// 如果是 string，假設它已經是 JSON，直接轉 []byte
		b = []byte(v)
	case []byte:
		// 如果已經是 []byte，直接用
		b = v
	default:
		// 其他類型（struct/map）再 Marshal
		var err error
		b, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}

	// 寫入一行
	_, err = f.Write(append(b, '\n'))
	return err
}
