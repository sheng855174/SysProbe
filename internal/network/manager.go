package network

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sysprobe/internal/config"
	"sysprobe/internal/monitor/cpu"
	"sysprobe/internal/monitor/disk"
	"sysprobe/internal/monitor/memory"
	"sysprobe/internal/monitor/network"
	logstream "sysprobe/internal/network/logstream"
	"sysprobe/internal/utils"
	"time"

	"google.golang.org/grpc"
)

type OffsetState struct {
	Offset    int64  `json:"offset"`
	Timestamp string `json:"timestamp"`
}

const connectedRetryTime = 30

func LoadNetwork(ctx context.Context, cfg config.Config) {
	utils.Log.Info("Network Manager starting...")
	for _, c := range cfg.Network.Category {
		prefix := transferCategory(c)
		if prefix == "" {
			continue
		}
		go func() {
			dir := cfg.Monitor.Data
			ignoreOlder := cfg.Network.IgnoreOlder
			stream := connectStream(cfg.Network)

			for {
				select {
				case <-ctx.Done():
					utils.Log.Info("[Network] STOP: context canceled.")
					return

				default:
					// 依照「檔名上的日期」挑最新三個檔
					logFiles := latestNByFilename(prefix, dir+"/"+prefix, ignoreOlder)
					for _, file := range logFiles {
						err := tailOneFile(ctx, file, cfg.Network, stream)
						if err != nil {
							utils.Log.Error("[Network] tail error:%v", err)
						}
					}
					// 每 30 秒重新取一次最新 3 檔
					time.Sleep(30 * time.Second)
				}
			}
		}()
	}
	// 等待所有 goroutine
	<-ctx.Done()
	utils.Log.Info("Netwrok started")
}

func transferCategory(category string) string {
	switch category {
	case "cpu":
		return cpu.Category
	case "disk":
		return disk.Category
	case "memory":
		return memory.Category
	case "netwrok":
		return network.Category
	}
	return ""
}

// ------------------------- 最新 N 檔案（依檔名日期排序） -------------------------
func latestNByFilename(prefix, dir string, n int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		utils.Log.Error("read dir fail:%v", err)
		return nil
	}

	type fileInfo struct {
		name string
		ts   time.Time
	}

	var files []fileInfo

	for _, e := range entries {
		name := e.Name()

		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// 去掉 prefix，例如 "network-2025-11-23" → "2025-11-23"
		dateStr := strings.TrimPrefix(name, prefix+"-")

		// 去掉副檔名（如果有）
		dateStr = strings.Split(dateStr, ".")[0]

		// 解析日期格式 YYYY-MM-DD
		ts, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// 檔名不合法 → 略過
			continue
		}

		files = append(files, fileInfo{
			name: filepath.Join(dir, name),
			ts:   ts,
		})
	}

	// 依日期排序（新 → 舊）
	sort.Slice(files, func(i, j int) bool {
		return files[i].ts.After(files[j].ts)
	})

	// 只取 N 個
	if len(files) > n {
		files = files[:n]
	}

	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.name
	}
	return out
}

// ------------------------- Tail 單個檔案 -------------------------
func tailOneFile(ctx context.Context, filePath string, cfg config.NetworkConfig, stream logstream.LogStreamer_StreamLogsClient) error {
	// 避免 offset 重覆加
	offsetFile := filePath
	if !strings.HasSuffix(filePath, ".offset") {
		offsetFile = filePath + ".offset"
	}

	// 讀取 offset 狀態
	lastOffset := loadOffsetState(offsetFile)

	// 打開檔案
	f, err := os.Open(filePath)
	if err != nil {
		utils.Log.Error("open %s fail: %v", filePath, err)
		return err
	}
	defer f.Close()

	// 從 offset 移動
	if _, err := f.Seek(lastOffset.Offset, io.SeekStart); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	var seq uint64 = 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					// 檔案讀完 → 等待下一輪外層迴圈或結束
					return nil
				}
				utils.Log.Error("file read error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			// 建立 event
			event := &logstream.LogEvent{
				Seq:       seq,
				Timestamp: time.Now().Format(time.RFC3339Nano),
				Source:    "SysProbe",
				Payload:   []byte(line),
			}

			// gRPC 發送
			for {
				if err := stream.Send(event); err != nil {
					utils.Log.Debug("send failed, reconnecting: %v", err)
					time.Sleep(time.Second)
					stream = connectStream(cfg)
					continue
				}
				break
			}

			// 更新 offset
			pos, _ := f.Seek(0, io.SeekCurrent)
			saveOffsetState(offsetFile, pos)
			seq++
		}
	}
}

// ------------------------- gRPC -------------------------
func connectStream(cfg config.NetworkConfig) logstream.LogStreamer_StreamLogsClient {
	for {
		conn, err := grpc.Dial(cfg.Host, grpc.WithInsecure())
		if err != nil {
			utils.Log.Error("dial failed, retrying: %v", err)
			time.Sleep(connectedRetryTime * time.Second)
			continue
		}

		client := logstream.NewLogStreamerClient(conn)

		stream, err := client.StreamLogs(context.Background())
		if err != nil {
			utils.Log.Error("stream failed, retrying: %v", err)
			time.Sleep(connectedRetryTime * time.Second)
			continue
		}

		utils.Log.Debug("gRPC connected")
		return stream
	}
}

// ------------------------- Offset State -------------------------
func loadOffsetState(path string) OffsetState {
	data, err := os.ReadFile(path)
	if err != nil {
		return OffsetState{Offset: 0}
	}
	var s OffsetState
	if err := json.Unmarshal(data, &s); err != nil {
		return OffsetState{Offset: 0}
	}
	return s
}

func saveOffsetState(path string, offset int64) {
	state := OffsetState{
		Offset:    offset,
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}

	b, _ := json.MarshalIndent(state, "", "  ")

	tmp := path + ".tmp"
	_ = os.WriteFile(tmp, b, 0644)
	_ = os.Rename(tmp, path)
}
