package httpadapter

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const traceIDKey = "trace_id"

func traceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = newTraceID()
		}
		c.Set(traceIDKey, traceID)
		c.Header("X-Trace-Id", traceID)
		c.Next()
	}
}

func traceID(c *gin.Context) string {
	if v, ok := c.Get(traceIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return newTraceID()
}

func newTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "trace"
	}
	return hex.EncodeToString(b)
}
