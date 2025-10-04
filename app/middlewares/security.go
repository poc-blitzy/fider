package middlewares

import (
	"fmt"
	"strings"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/web"
)

// Secure middleware is responsible for
// 1. Setting the HTTP Security Headers
// 2. Protecting from Host attacks
func Secure() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			cdnHost := env.Config.CDN.Host
			if cdnHost != "" && !env.IsSingleHostMode() {
				cdnHost = "*." + cdnHost
			}
			csp := fmt.Sprintf(web.CspPolicyTemplate, c.ContextID(), cdnHost)

			c.Response.Header().Set("Content-Security-Policy", strings.TrimSpace(csp))
			c.Response.Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response.Header().Set("X-Content-Type-Options", "nosniff")
			c.Response.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
			return next(c)
		}
	}
}

// Secure middleware is responsible for blocking CSRF attacks
func CSRF() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			var isWriteRequest = c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE"
			var hasBearerToken = strings.HasPrefix(c.Request.GetHeader("Authorization"), "Bearer ")
			
			// Exempt AJAX requests (JSON content-type) and Bearer token requests from CSRF validation
			// AJAX requests are safe because browsers enforce CORS
			// Bearer tokens are safe because they're not automatically sent like cookies
			if isWriteRequest && !c.IsAjax() && !hasBearerToken {
				return c.Forbidden()
			}
			return next(c)
		}
	}
}
