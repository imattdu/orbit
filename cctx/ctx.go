package cctx

import (
	"context"
	"time"
)

// ----------------- 内部类型 -----------------

type bagKeyType struct{}

var bagKey bagKeyType

// bag 是不可变语义的键值容器：每次写入时都会复制一份
type bag map[string]any

// 提取 bag（可能为 nil）
func bagFrom(ctx context.Context) bag {
	if b, ok := ctx.Value(bagKey).(bag); ok && b != nil {
		return b
	}
	return nil
}

// 深拷贝：仅针对 map[string]any 与 []any 做递归 copy，其它类型按值赋（引用类型由业务自控）
func deepCopy(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return deepCopyMap(x)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = deepCopy(x[i])
		}
		return out
	default:
		// 其它类型（如 struct 指针、slice/Map 非 any 元素等）按值传递
		// 若需要更深的值语义，请在调用方提供自定义的可复制类型。
		return v
	}
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopy(v)
	}
	return out
}

// ----------------- 对外 API -----------------

// New 用给定数据（深拷贝）创建一个携带 Bag 的 ctx；不会改变 parent。
func New(parent context.Context, data map[string]any) context.Context {
	cp := deepCopyMap(data)
	return context.WithValue(parent, bagKey, bag(cp))
}

// With 在现有 ctx 上写入一条 k/v，返回新 ctx（不可变）
func With(ctx context.Context, key string, val any) context.Context {
	old := bagFrom(ctx)
	newMap := make(map[string]any, len(old)+1)
	for k, v := range old {
		newMap[k] = v
	}
	newMap[key] = deepCopy(val)
	return context.WithValue(ctx, bagKey, bag(newMap))
}

// WithMany 一次写入多条键值（不可变）
func WithMany(ctx context.Context, kv map[string]any) context.Context {
	old := bagFrom(ctx)
	newMap := make(map[string]any, len(old)+len(kv))
	for k, v := range old {
		newMap[k] = v
	}
	for k, v := range kv {
		newMap[k] = deepCopy(v)
	}
	return context.WithValue(ctx, bagKey, bag(newMap))
}

// Get 读取一个键
func Get(ctx context.Context, key string) (any, bool) {
	if b := bagFrom(ctx); b != nil {
		v, ok := b[key]
		return v, ok
	}
	return nil, false
}

// GetAs 读取并断言为 T
func GetAs[T any](ctx context.Context, key string) (T, bool) {
	var zero T
	v, ok := Get(ctx, key)
	if !ok {
		return zero, false
	}
	tv, ok := v.(T)
	if !ok {
		return zero, false
	}
	return tv, true
}

func GetOrNewAs[T any](ctx context.Context, key string, fn func() T) T {
	rsp, ok := GetAs[T](ctx, key)
	if ok {
		return rsp
	}
	return fn()
}

// All 返回 Bag 的**深拷贝**
func All(ctx context.Context) map[string]any {
	if b := bagFrom(ctx); b != nil {
		return deepCopyMap(b)
	}
	return map[string]any{}
}

// AllAs 过滤出能断言为 T 的键值（返回新 map）
func AllAs[T any](ctx context.Context) map[string]T {
	res := make(map[string]T)
	if b := bagFrom(ctx); b != nil {
		for k, v := range b {
			if tv, ok := v.(T); ok {
				res[k] = tv
			}
		}
	}
	return res
}

// Clone 复制一个“独立”的 ctx：
// - 复制 Bag（深拷贝）
// - 保留原 ctx 的 deadline（相同时间点）
// - 原 ctx Done() 时，联动 cancel 新 ctx
// 返回 (newCtx, cancel)：业务应在合适时机调用 cancel() 释放计时器
func Clone(parent context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()

	var (
		newCtx context.Context
		cancel context.CancelFunc
	)
	if dl, ok := parent.Deadline(); ok {
		newCtx, cancel = context.WithDeadline(base, dl)
	} else {
		newCtx, cancel = context.WithCancel(base)
	}

	// 联动取消
	if parent.Done() != nil {
		go func() {
			select {
			case <-parent.Done():
				cancel()
			case <-newCtx.Done():
			}
		}()
	}
	// 若 parent 已取消，立即取消新 ctx
	if err := parent.Err(); err != nil {
		cancel()
	}

	// 复制 Bag
	if b := bagFrom(parent); b != nil {
		newCtx = context.WithValue(newCtx, bagKey, bag(deepCopyMap(b)))
	}
	return newCtx, cancel
}

// CloneWithNewTimeout 复制 ctx，但把 deadline 改为“剩余时长 + offset”（常用于独立下游调用）
// - 若 parent 有 deadline，则 newTimeout = time.Until(parent.Deadline()) + offset
// - 若 parent 无 deadline，则仅使用 offset（<=0 表示不设定新超时）
func CloneWithNewTimeout(parent context.Context, offset time.Duration) (context.Context, context.CancelFunc) {
	base := context.Background()

	remain := time.Duration(0)
	if dl, ok := parent.Deadline(); ok {
		remain = time.Until(dl)
		if remain < 0 {
			remain = 0
		}
	}
	total := remain + offset

	var (
		newCtx context.Context
		cancel context.CancelFunc
	)
	if total > 0 {
		newCtx, cancel = context.WithTimeout(base, total)
	} else {
		newCtx, cancel = context.WithCancel(base)
	}

	// 联动取消
	if parent.Done() != nil {
		go func() {
			select {
			case <-parent.Done():
				cancel()
			case <-newCtx.Done():
			}
		}()
	}
	if err := parent.Err(); err != nil {
		cancel()
	}

	// 复制 Bag
	if b := bagFrom(parent); b != nil {
		newCtx = context.WithValue(newCtx, bagKey, bag(deepCopyMap(b)))
	}
	return newCtx, cancel
}
