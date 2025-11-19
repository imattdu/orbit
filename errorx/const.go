package errorx

// CodeEntry 表示一个错误码 + 默认文案。
// 建议只在这里集中定义，业务用变量名，不直接写裸 code。
type CodeEntry struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// -------------------- 外部错误类型（调用方能感知的大类） --------------------

var (
	ExternalErrDefault = CodeEntry{Code: 1, Message: "未知错误"}
	ExternalErrSys     = CodeEntry{Code: 2, Message: "系统错误"}
	ExternalErrService = CodeEntry{Code: 4, Message: "服务错误"}
	ExternalErrBiz     = CodeEntry{Code: 5, Message: "业务错误"}
)

// -------------------- 服务类型（组件 / 服务） --------------------

var (
	ServiceTypeDefault = CodeEntry{Code: 1, Message: "unknown service type"}
	ServiceTypeBasic   = CodeEntry{Code: 4, Message: "组件"}
	ServiceTypeService = CodeEntry{Code: 5, Message: "服务"}
)

// -------------------- 服务枚举（示例，可按需扩展） --------------------

var (
	ServiceDefault = CodeEntry{Code: 1, Message: "未知服务"}
	ServiceMysql   = CodeEntry{Code: 4, Message: "mysql"}
	ServiceRedis   = CodeEntry{Code: 6, Message: "redis"}
	ServiceBaidu   = CodeEntry{Code: 91, Message: "baidu"}
	ServicePing    = CodeEntry{Code: 92, Message: "ping"} // 避免和 ServiceBaidu 重复
)

// -------------------- 错误类别（系统 / 业务） --------------------

var (
	ErrTypeSys = CodeEntry{Code: 4, Message: "系统错误"}
	ErrTypeBiz = CodeEntry{Code: 5, Message: "业务错误"}
)

// -------------------- 结果枚举 --------------------

var (
	Success = CodeEntry{Code: 0, Message: "success"}
	Failed  = CodeEntry{Code: 1, Message: "failed"}
)

// -------------------- 通用业务错误 --------------------

var (
	ErrDefault  = CodeEntry{Code: 1000, Message: "未知错误"}
	ErrNotFound = CodeEntry{Code: 404, Message: "not found"}
)
