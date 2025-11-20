package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type caller struct {
	file     string
	line     int
	funcName string
}

// -------------------- 调用方信息 --------------------

func getCaller() caller {
	// skip=0:getCaller, 1:encodeLog/log, 2:对外 Info/Debug/Error 调用者
	pc, file, line, ok := runtime.Caller(2)
	file = trimFilePath(file)
	funcName := "unknown"
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = trimFuncName(fn.Name())
		}
	}
	return caller{
		file:     file,
		line:     line,
		funcName: funcName,
	}
}

var (
	modRootOnce sync.Once
	modRoot     string
)

func getModRoot(fullPath string) string {
	modRootOnce.Do(func() {
		m, err := findGoModRoot(fullPath)
		if err == nil {
			modRoot = m
		}
	})
	return modRoot
}

// 把 /Users/xxx/project/orbit/logx/logger.go -> orbit/logx/logger.go
func trimFilePath(fullPath string) string {
	if fullPath == "" {
		return ""
	}
	if root := getModRoot(fullPath); root != "" {
		if rel, err := filepath.Rel(root, fullPath); err == nil {
			return rel
		}
	}
	_, short := filepath.Split(fullPath)
	return short
}

func findGoModRoot(start string) (string, error) {
	dir := filepath.Dir(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}

// github.com/imattdu/orbit/logx.Info -> Info
// github.com/imattdu/orbit/service.(*Server).Serve -> (*Server).Serve
func trimFuncName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.Index(name, "."); idx >= 0 && idx+1 < len(name) {
		name = name[idx+1:]
	}
	return name
}
