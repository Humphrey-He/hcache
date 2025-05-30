package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noobtrump/hcache/pkg/cache"
)

// RequestLogger returns a middleware that logs request information
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Log request details
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		fmt.Printf("[%s] %s %s %d %s %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			method,
			path,
			statusCode,
			latency,
			clientIP,
		)
	}
}

// CacheMetrics returns a middleware that adds cache metrics to the response headers
func CacheMetrics(cacheInstance cache.ICache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()

		// Get cache stats
		stats, err := cacheInstance.Stats(c)
		if err != nil {
			return
		}

		// Add cache metrics to response headers
		c.Header("X-Cache-Hits", fmt.Sprintf("%d", stats.Hits))
		c.Header("X-Cache-Misses", fmt.Sprintf("%d", stats.Misses))

		// 计算命中率
		hitRatio := 0.0
		if stats.Hits+stats.Misses > 0 {
			hitRatio = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
		}
		c.Header("X-Cache-Hit-Ratio", fmt.Sprintf("%.2f", hitRatio))

		c.Header("X-Cache-Entries", fmt.Sprintf("%d", stats.EntryCount))
		c.Header("X-Cache-Size", fmt.Sprintf("%d", stats.Size))
	}
}
