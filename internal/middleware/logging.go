package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger(appEnv string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		if appEnv == "production" || appEnv == "staging" {
			writeJSONLog(map[string]any{
				"level":      "info",
				"message":    "http_request",
				"request_id": GetRequestID(c),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"status":     c.Writer.Status(),
				"latency_ms": float64(latency.Microseconds()) / 1000.0,
				"client_ip":  c.ClientIP(),
			})
			return
		}

		log.Printf("request_id=%s method=%s path=%s status=%d latency=%s client_ip=%s",
			GetRequestID(c), c.Request.Method, c.Request.URL.Path, c.Writer.Status(), latency, c.ClientIP())
	}
}

func writeJSONLog(entry map[string]any) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("log_marshal_error=%v", err)
		return
	}
	log.Print(string(data))
}
