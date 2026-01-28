package middleware

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// debugMode determines if detailed error logging is enabled
var debugMode = os.Getenv("DEBUG") == "true"

// RequestLogger logs HTTP requests with timing information
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		log.Printf("[%s] %d %s %s %s %v",
			time.Now().Format("2006/01/02 15:04:05"),
			status,
			method,
			path,
			clientIP,
			latency,
		)
	}
}

// Recovery middleware recovers from panics and logs the error
// In debug mode, logs full panic details; otherwise logs generic message
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log generic message by default to avoid leaking internals
				log.Println("[PANIC] Server recovered from panic")

				// Only log detailed error in debug mode
				if debugMode {
					log.Printf("[PANIC DEBUG] %v", err)
				}

				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
