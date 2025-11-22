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

	mu sync.Mutex

	infoFile *os.File
	warnFile *os.File

	infoSize int64
	warnSize int64

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

	now := time.Now()

	// 先打开 info/warn 两个文件
	h.mu.Lock()
	if err := h.rotateIfNeededLocked(now); err != nil {
		h.mu.Unlock()
		return nil, err
	}
	h.mu.Unlock()

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
	_ = attrs
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

// writeRecord 把 Record 编码成 JSON 一行，写入 info/warn 文件 + 控制台
func (h *handler) writeRecord(r slog.Record) error {
	// 先构造 JSON 行，减少持锁时间
	data := make(map[string]any, 16)
	data["ts"] = r.Time.Format(time.RFC3339Nano)
	data["level"] = r.Level.String()

	r.Attrs(func(a slog.Attr) bool {
		v := a.Value
		data[a.Key] = v.Any()
		return true
	})

	lineBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	line := string(lineBytes) + "\n"

	now := time.Now()

	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.rotateIfNeededLocked(now); err != nil {
		return err
	}

	// 选择 info / warn 文件
	var f *os.File
	if r.Level >= slog.LevelWarn {
		f = h.warnFile
	} else {
		f = h.infoFile
	}
	if f != nil {
		n, err := f.WriteString(line)
		if err != nil {
			return err
		}
		if r.Level >= slog.LevelWarn {
			h.warnSize += int64(n)
		} else {
			h.infoSize += int64(n)
		}
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

// rotateIfNeededLocked 在已上锁的情况下，根据配置判断是否需要切分 info/warn 文件
func (h *handler) rotateIfNeededLocked(now time.Time) error {
	needNew := false
	needWarnNew := false
	switch *h.cfg.Rotate {
	case RotateHourly:
		// 按小时切
		hour := now.Truncate(time.Hour)
		if h.curHr.IsZero() || !hour.Equal(h.curHr) {
			needNew = true
			needWarnNew = true
			h.curHr = hour
			h.infoSize = 0
			h.warnSize = 0
		}
		if h.infoFile == nil {
			needNew = true
		}
		if h.warnFile == nil {
			needWarnNew = true
		}
	case RotateSize:
		if h.infoFile == nil {
			needNew = true
		}
		if h.warnFile == nil {
			needWarnNew = true
		}
		if h.cfg.MaxFileSizeMB > 0 {
			limit := int64(h.cfg.MaxFileSizeMB) * 1024 * 1024
			if h.infoSize >= limit {
				needNew = true
				h.infoSize = 0
			}
			if h.warnSize >= limit {
				needWarnNew = true
				h.warnSize = 0
			}
		}
	}

	if !needNew && !needWarnNew {
		return nil
	}

	// 关闭旧文件
	if needNew && h.infoFile != nil {
		_ = h.infoFile.Close()
		h.infoFile = nil
	}
	if needWarnNew && h.warnFile != nil {
		_ = h.warnFile.Close()
		h.warnFile = nil
	}

	if err := os.MkdirAll(h.cfg.LogDir, 0o755); err != nil {
		return err
	}
	// 打开新的 info / warn 文件
	if needNew {
		infoName := h.buildFilename(now, false)
		infoFile, err := os.OpenFile(infoName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		h.infoFile = infoFile
		infoLink := filepath.Join(h.cfg.LogDir, h.cfg.AppName+".log")
		_ = os.Remove(infoLink)
		_ = os.Symlink(filepath.Base(infoName), infoLink)
	}

	if needWarnNew {
		warnName := h.buildFilename(now, true)
		warnFile, err := os.OpenFile(warnName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		h.warnFile = warnFile
		warnLink := filepath.Join(h.cfg.LogDir, h.cfg.AppName+".wf.log")

		_ = os.Remove(warnLink)
		_ = os.Symlink(filepath.Base(warnName), warnLink)
	}

	// 清理旧文件
	if h.cfg.MaxBackups > 0 {
		h.cleanupOldFiles(h.infoPrefix())
		h.cleanupOldFiles(h.warnPrefix())
	}
	return nil
}

// buildFilename 构造 info / warn 日志文件名
func (h *handler) buildFilename(now time.Time, warn bool) string {
	var ts string
	if *h.cfg.Rotate == RotateSize {
		// 按大小切时，也带上日期，方便排查
		ts = now.Format("20060102150405") // 到秒
	} else {
		ts = now.Format("2006010215") // 到小时
	}

	name := h.cfg.AppName
	if warn {
		// warn 文件加 .wf 前缀，和常见 app.wf.log 习惯一致
		return filepath.Join(h.cfg.LogDir, fmt.Sprintf("%s.wf-%s.log", name, ts))
	}
	return filepath.Join(h.cfg.LogDir, fmt.Sprintf("%s-%s.log", name, ts))
}

func (h *handler) infoPrefix() string {
	return h.cfg.AppName + "-"
}

func (h *handler) warnPrefix() string {
	return h.cfg.AppName + ".wf-"
}

// cleanupOldFiles 只清理指定前缀的日志文件（info 或 warn）
func (h *handler) cleanupOldFiles(prefix string) {
	entries, err := os.ReadDir(h.cfg.LogDir)
	if err != nil {
		log.Println("cleanupOldFiles ReadDir error:", err)
		return
	}

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
