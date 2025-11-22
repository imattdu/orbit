package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/imattdu/orbit/logx"

	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	body *bytes.Buffer
	gin.ResponseWriter
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

var accessLogger logx.Logger

func InitAccessLogger(logger logx.Logger) error {
	if logger != nil {
		accessLogger = logger
		return nil
	}

	l, err := logx.New(logx.Config{
		AppName:    "access",
		Level:      slog.LevelInfo,
		LogDir:     "logs",
		MaxBackups: 24,
	})
	if err != nil {
		accessLogger = l
	}
	return err
}

// AccessMiddleware 简单访问日志
func AccessMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取body
		req := ctx.Request
		c := req.Context()
		logMap := map[string]interface{}{
			logx.Remote: req.RemoteAddr,
			logx.Method: req.Method,
			logx.Path:   req.URL.Path,
			logx.Query:  req.URL.RawQuery,
		}
		reqBodyBytes, err := ctx.GetRawData()
		if err != nil {
			_ = ctx.AbortWithError(http.StatusInternalServerError, err)
			logMap[logx.Err] = err.Error()
			logMap[logx.Msg] = "GetRawData failed"
			accessLogger.Warn(c, logx.TagUndef, logMap)
			return
		}

		// 重置HTTP请求体的偏移量
		ctx.Request.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
		var reqBody interface{}
		_ = json.Unmarshal(reqBodyBytes, &reqBody)
		accessLogger.Info(c, logx.TagRequestIn, logMap)
		logMap[logx.Body] = reqBody

		// 捕捉响应
		writer := &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
		ctx.Writer = writer
		start := time.Now()
		ctx.Next()

		logMap[logx.Response] = writer.body.String()
		logMap[logx.Cost] = time.Since(start).Milliseconds()
		accessLogger.Info(c, logx.TagRequestOut, logMap)
	}
}
