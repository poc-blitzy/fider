package middlewares

import (
	"net"
	"path"
	"strings"

	"net/http"
	"net/url"

	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/web"
)

// Tenant adds either SingleTenant or MultiTenant to the pipeline
func Tenant() web.MiddlewareFunc {
	if env.IsSingleHostMode() {
		return SingleTenant()
	}
	return MultiTenant()
}

// SingleTenant inject default tenant into current context
func SingleTenant() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			firstTenant := &query.GetFirstTenant{}
			err := bus.Dispatch(c, firstTenant)
			if err != nil && errors.Cause(err) != app.ErrNotFound {
				return c.Failure(err)
			}

			if firstTenant.Result != nil && !firstTenant.Result.IsDisabled() {
				c.SetTenant(firstTenant.Result)
			}

			return next(c)
		}
	}
}

// MultiTenant extract tenant information from hostname or X-Tenant-ID header and inject it into current context
// Resolution priority: X-Tenant-ID header > hostname (subdomain/CNAME)
func MultiTenant() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			// Priority 1: Try to resolve tenant from X-Tenant-ID header (for cross-origin SPA clients)
			// This enables frontend applications on different origins to explicitly specify tenant context
			tenantID := c.Request.GetHeader("X-Tenant-ID")
			var domain string
			
			if tenantID != "" {
				// Header present: use the provided tenant identifier for resolution
				// Strip port number if present (e.g., "acme:8080" -> "acme")
				// Strip port number if present using net.SplitHostPort
				host, _, err := net.SplitHostPort(tenantID)
				if err != nil {
					// No port present, use the original value
					domain = tenantID
				} else {
					// Port was present, use the host part
					domain = host
				}
			} else {
				// Header absent: fall back to hostname-based resolution (subdomain or CNAME)
				// This preserves backward compatibility with existing same-origin clients
				domain = c.Request.URL.Hostname()
			}

			// Query tenant by domain (works for subdomain, CNAME, or explicit header value)
			byDomain := &query.GetTenantByDomain{Domain: domain}
			err := bus.Dispatch(c, byDomain)
			if err != nil && errors.Cause(err) != app.ErrNotFound {
				return c.Failure(err)
			}

			// Set tenant in context if found and active
			if byDomain.Result != nil && !byDomain.Result.IsDisabled() {
				c.SetTenant(byDomain.Result)

				// Canonical URL handling: only applies to CNAME tenants accessed via non-AJAX requests
				// This redirects users to the canonical domain if accessing via alternate domain
				if byDomain.Result.CNAME != "" && !c.IsAjax() {
					baseURL := web.TenantBaseURL(c, byDomain.Result)
					if baseURL != c.BaseURL() {
						link := baseURL + c.Request.URL.RequestURI()
						c.SetCanonicalURL(link)
					}
				}
			}

			return next(c)
		}
	}
}

// RequireTenant returns 404 if tenant is not available
func RequireTenant() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			tenant := c.Tenant()
			if tenant == nil {
				if env.IsSingleHostMode() {
					return c.Redirect("/signup")
				}
				return c.NotFound()
			}

			return next(c)
		}
	}
}

// BlockPendingTenants blocks requests for pending tenants
func BlockPendingTenants() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if c.Tenant().Status == enum.TenantPending {
				return c.Page(http.StatusOK, web.Props{
					Page:        "SignUp/PendingActivation.page",
					Title:       "Pending Activation",
					Description: "We sent you a confirmation email with a link to activate your site. Please check your inbox to activate it.",
				})
			}
			return next(c)
		}
	}
}

// CheckTenantPrivacy blocks requests of unauthenticated users for private tenants
func CheckTenantPrivacy() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if c.Tenant().IsPrivate && !c.IsAuthenticated() {
				parsedURL, err := url.Parse(c.Request.URL.String())
				if err != nil {
					return c.Failure(err)
				}

				cleanPath := path.Clean(parsedURL.RequestURI())
				redirectTarget := ""
				if strings.HasPrefix(cleanPath, "/") &&
					cleanPath != "/" &&
					!strings.HasPrefix(cleanPath, "/signin") &&
					!strings.HasPrefix(cleanPath, "/signout") &&
					!strings.Contains(cleanPath, "..") {
						redirectTarget = cleanPath
				}

				if redirectTarget != "" {
					return c.Redirect("/signin?redirect=" + url.QueryEscape(redirectTarget))
				}
				return c.Redirect("/signin")
			}
			return next(c)
		}
	}
}

// BlockLockedTenants blocks requests on locked tenants as they are in read-only mode
func BlockLockedTenants() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if c.Tenant().Status == enum.TenantLocked {

				// Only API operations are blocked, so it's ok to always return a JSON
				return c.JSON(http.StatusPaymentRequired, web.Map{})
			}
			return next(c)
		}
	}
}
