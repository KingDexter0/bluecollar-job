package middleware

import (
	"fmt"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	requests uint64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		atomic.AddUint64(&m.requests, 1)
		c.Next()
	}
}

func (m *Metrics) Handler(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4")
	c.String(200, fmt.Sprintf("# HELP bluecollar_http_requests_total Total HTTP requests handled.\n# TYPE bluecollar_http_requests_total counter\nbluecollar_http_requests_total %d\n", atomic.LoadUint64(&m.requests)))
}
