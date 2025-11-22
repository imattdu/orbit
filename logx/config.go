package logx

import "log/slog"

type RotateMode int

const (
	RotateHourly RotateMode = iota // 按小时切
	RotateSize                     // 按大小切
)

type Config struct {
	AppName string     // 应用名，用于文件名前缀
	Level   slog.Level // 最小日志级别

	LogDir string // 日志目录

	ConsoleEnabled bool // 是否输出到控制台
	ConsoleColored bool // 控制台是否彩色输出

	Rotate *RotateMode // 滚动模式

	// RotateSize 模式用：超过 size 就切新文件
	MaxFileSizeMB int
	// 最多保留多少个历史文件（按修改时间排序）
	MaxBackups int

	// 异步队列大小（<=0 使用默认 10000）
	QueueSize int
}
