package logx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type handler struct {
	cfg Config

	mu    sync.Mutex
	file  *os.File
	size  int64
	curHr time.Time // RotateHourly 使用：当前小时

	entries chan slog.Record
}

func newHandler(cfg Config) (slog.Handler, error) {
	if cfg.AppName == "" {
		cfg.AppName = "app"
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 10000
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "."
	}

	h := &handler{
		cfg:     cfg,
		entries: make(chan slog.Record, cfg.QueueSize),
	}

	// 先打开一个文件（根据当前时间）
	now := time.Now()
	if err := h.rotateIfNeeded(now); err != nil {
		return nil, err
	}

	go h.writeLoop()
	return h, nil
}

// Enabled 满足 slog.Handler 接口
func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.cfg.Level
}

// Handle 只负责把 Record 推入异步队列，队列满则丢（不阻塞业务）
func (h *handler) Handle(_ context.Context, r slog.Record) error {
	rr := r.Clone()
	select {
	case h.entries <- rr:
	default:
		log.Println("log queue full, drop log")
	}
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 简化：忽略 WithAttrs，所有 Attr 都由上层 encodeLog 提供
	return h
}

func (h *handler) WithGroup(name string) slog.Handler {
	// 简化：忽略分组
	_ = name
	return h
}

// 异步写 loop
func (h *handler) writeLoop() {
	for rec := range h.entries {
		if err := h.writeRecord(rec); err != nil {
			log.Println("write log failed:", err)
		}
	}
}

// writeRecord 把 Record 编码成 JSON 一行，写入文件 + 控制台
func (h *handler) writeRecord(r slog.Record) error {
	now := time.Now()
	if err := h.rotateIfNeeded(now); err != nil {
		return err
	}

	// 构造 JSON 行
	data := make(map[string]any, 16)
	data["ts"] = r.Time.Format(time.RFC3339Nano)
	data["level"] = r.Level.String()

	r.Attrs(func(a slog.Attr) bool {
		// 兼容所有 Go 版本，不使用 Resolve()
		av := a.Value
		data[a.Key] = av.Any()
		return true
	})

	lineBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	line := string(lineBytes) + "\n"

	// 写文件
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		n, err := h.file.WriteString(line)
		if err != nil {
			return err
		}
		h.size += int64(n)
	}

	// 控制台输出
	if h.cfg.ConsoleEnabled {
		if h.cfg.ConsoleColored {
			fmt.Print(h.colorLine(r, line))
		} else {
			fmt.Print(line)
		}
	}

	return nil
}

// rotateIfNeeded 根据配置判断是否需要切分文件
func (h *handler) rotateIfNeeded(now time.Time) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	needNew := false

	switch h.cfg.Rotate {
	case RotateHourly:
		hour := now.Truncate(time.Hour)
		if h.file == nil || h.curHr.IsZero() || !hour.Equal(h.curHr) {
			needNew = true
			h.curHr = hour
			h.size = 0
		}
	case RotateSize:
		if h.file == nil {
			needNew = true
		} else if h.cfg.MaxFileSizeMB > 0 {
			limit := int64(h.cfg.MaxFileSizeMB) * 1024 * 1024
			if h.size >= limit {
				needNew = true
				h.size = 0
			}
		}
	default:
		// 默认按小时
		hour := now.Truncate(time.Hour)
		if h.file == nil || h.curHr.IsZero() || !hour.Equal(h.curHr) {
			needNew = true
			h.curHr = hour
			h.size = 0
		}
	}

	if !needNew {
		return nil
	}

	if h.file != nil {
		_ = h.file.Close()
	}

	filename := h.buildFilename(now)
	if err := os.MkdirAll(h.cfg.LogDir, 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	h.file = f

	// 更新软链：{AppName}.log -> 当前文件
	linkPath := filepath.Join(h.cfg.LogDir, h.cfg.AppName+".log")
	_ = os.Remove(linkPath)
	_ = os.Symlink(filepath.Base(filename), linkPath)

	// 清理旧文件
	if h.cfg.MaxBackups > 0 {
		h.cleanupOldFiles()
	}

	return nil
}

// buildFilename 构造日志文件名
func (h *handler) buildFilename(now time.Time) string {
	var ts string
	if h.cfg.Rotate == RotateSize {
		ts = now.Format("20060102150405") // 到秒
	} else {
		ts = now.Format("2006010215") // 到小时
	}
	return filepath.Join(h.cfg.LogDir, fmt.Sprintf("%s-%s.log", h.cfg.AppName, ts))
}

// cleanupOldFiles 按修改时间排序，只保留最新 MaxBackups 个
func (h *handler) cleanupOldFiles() {
	entries, err := os.ReadDir(h.cfg.LogDir)
	if err != nil {
		log.Println("cleanupOldFiles ReadDir error:", err)
		return
	}

	prefix := h.cfg.AppName + "-"
	suffix := ".log"

	type fi struct {
		name string
		t    time.Time
	}

	files := make([]fi, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fi{
			name: filepath.Join(h.cfg.LogDir, name),
			t:    info.ModTime(),
		})
	}

	if len(files) <= h.cfg.MaxBackups {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].t.After(files[j].t) // 新的在前
	})

	for _, f := range files[h.cfg.MaxBackups:] {
		_ = os.Remove(f.name)
	}
}

// colorLine 简单根据 level 加点前缀颜色（用现成的 JSON 行）
func (h *handler) colorLine(r slog.Record, line string) string {
	level := r.Level.String()
	switch r.Level {
	case slog.LevelDebug:
		return "\033[36m[DEBUG]\033[0m " + line
	case slog.LevelInfo:
		return "\033[32m[INFO ]\033[0m " + line
	case slog.LevelWarn:
		return "\033[33m[WARN ]\033[0m " + line
	case slog.LevelError:
		return "\033[31m[ERROR]\033[0m " + line
	default:
		return "[" + level + "] " + line
	}
}
