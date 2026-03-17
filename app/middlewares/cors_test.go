package middlewares_test

import (
	"net/http"
	"testing"

	"github.com/getfider/fider/app/middlewares"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

// TestCORS_MatchingOrigin_SetsHeaders verifies that when the request Origin
// matches the ALLOWED_ORIGINS allowlist, the CORS middleware sets the correct
// Access-Control-Allow-Origin, Access-Control-Allow-Credentials, and Vary headers.
func TestCORS_MatchingOrigin_SetsHeaders(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001"
	defer func() { env.Config.AllowedOrigins = "" }()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	server.AddHeader("Origin", "http://localhost:3001")
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("http://localhost:3001")
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("true")
	Expect(response.Header().Get("Vary")).Equals("Origin")
}

// TestCORS_NonMatchingOrigin_NoHeaders verifies that when the request Origin
// does NOT match the ALLOWED_ORIGINS allowlist, no CORS headers are set.
// The browser will block the response on the client side.
func TestCORS_NonMatchingOrigin_NoHeaders(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001"
	defer func() { env.Config.AllowedOrigins = "" }()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	server.AddHeader("Origin", "http://evil.example.com")
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("")
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("")
}

// TestCORS_PreflightRequest_Returns204 verifies that an OPTIONS preflight request
// from an allowed origin is short-circuited with HTTP 204 and the correct
// Access-Control-Allow-Methods, Access-Control-Allow-Headers, and Access-Control-Max-Age
// headers are returned.
func TestCORS_PreflightRequest_Returns204(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001"
	defer func() { env.Config.AllowedOrigins = "" }()

	server := mock.NewServer()
	// Pre-middleware to set method to OPTIONS (runs before CORS middleware)
	// This is necessary because mock.NewServer() creates a GET request by default
	// and the context field is unexported. Middleware[0] runs first in the Execute
	// wrapping order, so this method setter executes before CORS.
	server.Use(func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			c.Request.Method = "OPTIONS"
			return next(c)
		}
	})
	server.Use(middlewares.CORS())
	server.AddHeader("Origin", "http://localhost:3001")

	// This handler should never execute because CORS short-circuits OPTIONS requests
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusNoContent)
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("http://localhost:3001")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).Equals("GET, POST, PUT, DELETE, PATCH, OPTIONS")
	Expect(response.Header().Get("Access-Control-Allow-Headers")).Equals("Content-Type, Authorization, X-Csrf-Token")
	Expect(response.Header().Get("Access-Control-Max-Age")).Equals("86400")
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("true")
}

// TestCORS_MultipleAllowedOrigins verifies that when multiple origins are
// configured in ALLOWED_ORIGINS (comma-separated), each allowed origin
// receives the correct CORS headers when making requests.
func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001,https://app.example.com"
	defer func() { env.Config.AllowedOrigins = "" }()

	// First request: origin http://localhost:3001
	server1 := mock.NewServer()
	server1.Use(middlewares.CORS())
	server1.AddHeader("Origin", "http://localhost:3001")
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	status1, response1 := server1.Execute(handler)
	Expect(status1).Equals(http.StatusOK)
	Expect(response1.Header().Get("Access-Control-Allow-Origin")).Equals("http://localhost:3001")

	// Second request: origin https://app.example.com
	server2 := mock.NewServer()
	server2.Use(middlewares.CORS())
	server2.AddHeader("Origin", "https://app.example.com")

	status2, response2 := server2.Execute(handler)
	Expect(status2).Equals(http.StatusOK)
	Expect(response2.Header().Get("Access-Control-Allow-Origin")).Equals("https://app.example.com")
}

// TestCORS_NoOriginHeader_NoHeaders verifies that when no Origin header is
// present on the request, no CORS headers are set. This is a same-origin
// request and does not require CORS handling.
func TestCORS_NoOriginHeader_NoHeaders(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001"
	defer func() { env.Config.AllowedOrigins = "" }()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	// No Origin header added intentionally
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("")
}

// TestCORS_VaryOriginHeader verifies that the Vary: Origin header is always
// set when the CORS middleware processes a matching origin. This is required
// so that intermediate caches do not serve a response with the wrong
// Access-Control-Allow-Origin for different origins.
func TestCORS_VaryOriginHeader(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001"
	defer func() { env.Config.AllowedOrigins = "" }()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	server.AddHeader("Origin", "http://localhost:3001")
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	_, response := server.Execute(handler)

	Expect(response.Header().Get("Vary")).Equals("Origin")
}
