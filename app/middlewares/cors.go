package middlewares

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/getfider/fider/app/pkg/web"
)

// CORS adds Cross-Origin Resource Sharing response headers
func CORS() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			// Read configuration from environment variables (supports dynamic test config)
			allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
			allowCredentials := os.Getenv("CORS_ALLOW_CREDENTIALS") == "true"
			allowedMethods := os.Getenv("ALLOWED_METHODS")
			if allowedMethods == "" {
				allowedMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
			}
			allowedHeaders := os.Getenv("ALLOWED_HEADERS")
			if allowedHeaders == "" {
				allowedHeaders = "Authorization,Content-Type,X-Tenant-ID"
			}
			maxAgeStr := os.Getenv("CORS_MAX_AGE")
			maxAge := 600 // default 10 minutes
			if maxAgeStr != "" {
				if parsed, err := strconv.Atoi(maxAgeStr); err == nil && parsed > 0 {
					maxAge = parsed
				}
			}

			// Parse allowed origins
			var allowedOrigins []string
			if allowedOriginsStr != "" {
				for _, origin := range strings.Split(allowedOriginsStr, ",") {
					allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
				}
			}

			// Debug: Log request details and headers
			fmt.Printf("[CORS DEBUG] ===== CORS Middleware Processing =====\n")
			fmt.Printf("[CORS DEBUG] Request Method: %s\n", c.Request.Method)
			fmt.Printf("[CORS DEBUG] Request URL: %s\n", c.Request.URL.String())
			fmt.Printf("[CORS DEBUG] Allowed Origins: %v\n", allowedOrigins)
			fmt.Printf("[CORS DEBUG] Allow Credentials: %v\n", allowCredentials)

			origin := c.Request.GetHeader("Origin")
			fmt.Printf("[CORS DEBUG] Extracted Origin Header: '%s'\n", origin)


			
			// Validate origin against allow-list
			allowedOrigin := ""
			if origin != "" {
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						allowedOrigin = origin
						break
					}
				}
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				c.Response.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				
				if allowCredentials {
					c.Response.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				
				// Set these headers on all requests for consistency
				c.Response.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				c.Response.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			}

			// Handle preflight OPTIONS requests
			if c.Request.Method == "OPTIONS" {
				if allowedOrigin != "" {
					// Set Max-Age header only for preflight responses
					c.Response.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
				}
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}
