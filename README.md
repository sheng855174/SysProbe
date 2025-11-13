# SysProbe

## Go Service Framework  

本專案使用 Golang 1.25.4 開發，提供一個模組化的系統架構，包含：

- **服務管理器**：負責註冊與監控所有子服務
- **資料蒐集器**：可模組化開關、支援多平台（Linux / Windows）
- **網路模組**：提供通訊、API、Socket 等功能

專案強調 **跨平台、可擴充、易維護**。

## 專案結構  
```
.
├── cmd/
│   ├── service/         # 主程式：服務註冊與運行
│   ├── monitor/       # 資料蒐集器：統一管理各模組
│   └── network/         # 網路模組：API / 通訊邏輯
│
├── internal/
│   ├── utils/
│   │   ├── log.go       # 共用工具
│   ├── config/          # YAML 設定載入與解析
│   ├── registry/        # 服務註冊中心
│   └── monitor/
│       ├── network/     # 網路資料（流量、port...）
│       ├── disk/        # 硬碟資料（大小、IO量...）
│       ├── cpu/         # CPU 使用率與負載
│       ├── config.go    # Monitor 模組設定與開關
│       └── manager.go   # Monitor 管理與協調
├── config.yml           # 使用者可編輯的設定檔
├── go.mod
├── go.sum
└── README.md
```

