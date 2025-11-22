package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/imattdu/orbit/tracex"
)

// TraceMiddleware 生成/透传 trace，写入 ctx 和响应头
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, _, _ := tracex.ExtractRemoteSpan(context.Background(), c.Request.Header, "req")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
