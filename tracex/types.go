package tracex

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Span struct {
	TraceID string            `json:"trace_id"`
	SpanID  string            `json:"span_id"`
	Parent  string            `json:"parent_span_id,omitempty"`
	Name    string            `json:"name,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`

	Start time.Time      `json:"start"`
	End   time.Time      `json:"end"`
	Err   error          `json:"-"`
	raw   map[string]any // 预留扩展（比如耗时、额外字段）
}

// -------------------- ID 生成 --------------------

// newID 生成 128 bit 的随机 ID（32 位 hex）
func newID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return fallbackID()
	}
	return hex.EncodeToString(b[:])
}

func fallbackID() string {
	var b [16]byte
	for i := range b {
		b[i] = byte(i*31 + 17)
	}
	return hex.EncodeToString(b[:])
}
