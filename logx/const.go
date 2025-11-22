package logx

const (
	TagUndef       = "undef"
	TagRequestIn   = "request_in"
	TagRequestOut  = "request_out"
	TagHttpSuccess = "http_success"
	TagHttpFailure = "http_failure"

	Cost = "cost"
	Msg  = "msg"
	Err  = "err"

	Remote   = "remote"
	Method   = "method"
	URL      = "url"
	Path     = "path"
	Query    = "query"
	Request  = "request"
	Body     = "body"
	Response = "response"

	Attempt     = "attempt"
	Attempts    = "attempts"
	MaxAttempts = "max_attempts"
)
