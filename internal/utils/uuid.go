package utils

import (
	"encoding/json"
	"os"
	"time"

	"github.com/google/uuid"
)

type UUIDInfo struct {
	UUID       string `json:"uuid"`
	CreateTime string `json:"createTime"`
}

func InitUUID(path string) (UUIDInfo, error) {
	var info UUIDInfo

	// 若資料夾不存在，建立它
	if err := os.MkdirAll(path, 0755); err != nil {
		return info, err
	}

	filePath := path + "/uuid.json"

	// 如果 uuid.json 存在 -> 直接讀取
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return info, err
		}
		err = json.Unmarshal(data, &info)
		return info, err
	}

	// 若不存在 → 建立新 UUID
	info = UUIDInfo{
		UUID:       uuid.NewString(),
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return info, err
	}

	err = os.WriteFile(filePath, data, 0644)
	return info, err
}
