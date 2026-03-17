package middlewares

import (
	"net/http"
	"strings"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/web"
)

// CORS adds Cross-Origin Resource Sharing response headers.
// It reads the ALLOWED_ORIGINS env var (via env.Config.AllowedOrigins),
// validates the request Origin against the allowlist, and sets appropriate
// Access-Control-* headers. Wildcard origins are not used because
// Access-Control-Allow-Credentials: true prohibits wildcard per CORS spec.
func CORS() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			origin := c.Request.GetHeader("Origin")
			if origin == "" {
				// No Origin header — not a cross-origin request; skip CORS headers
				return next(c)
			}

			// Parse allowed origins from env config (comma-separated)
			allowedOrigins := env.Config.AllowedOrigins
			if allowedOrigins == "" {
				// No allowed origins configured — skip CORS headers
				return next(c)
			}

			// Check if the request origin is in the allowlist
			originAllowed := false
			for _, allowed := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(allowed) == origin {
					originAllowed = true
					break
				}
			}

			if !originAllowed {
				// Origin not in allowlist — proceed without CORS headers
				// The browser will block the response on the client side
				return next(c)
			}

			// Origin is allowed — set CORS response headers
			c.Response.Header().Set("Access-Control-Allow-Origin", origin)
			c.Response.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Response.Header().Set("Vary", "Origin")

			// Handle preflight OPTIONS requests
			if c.Request.Method == "OPTIONS" {
				c.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				c.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Csrf-Token")
				c.Response.Header().Set("Access-Control-Max-Age", "86400")
				// Short-circuit: return 204 without calling next middleware
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}
