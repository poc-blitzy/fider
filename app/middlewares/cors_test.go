package middlewares_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/getfider/fider/app/middlewares"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

func TestCORS_WithAllowedOrigin(t *testing.T) {
	RegisterT(t)

	// Configure allowed origins for testing
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://staging.app.example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	// Test with allowed origin
	server.AddHeader("Origin", "https://app.example.com")
	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("https://app.example.com")
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("true")
}

func TestCORS_WithDisallowedOrigin(t *testing.T) {
	RegisterT(t)

	// Configure allowed origins for testing
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	// Test with disallowed origin
	server.AddHeader("Origin", "https://malicious.com")
	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	// Origin header should not be set for disallowed origins
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("")
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("")
}

func TestCORS_PreflightRequest(t *testing.T) {
	RegisterT(t)

	// Configure CORS settings
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	os.Setenv("ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	os.Setenv("ALLOWED_HEADERS", "Authorization,Content-Type,X-Tenant-ID")
	defer func() {
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("ALLOWED_METHODS")
		os.Unsetenv("ALLOWED_HEADERS")
	}()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	// Simulate preflight OPTIONS request
	server.AddHeader("Origin", "https://app.example.com")
	server.AddHeader("Access-Control-Request-Method", "POST")
	server.AddHeader("Access-Control-Request-Headers", "Authorization,Content-Type")
	status, response := server.ExecuteAsOptions(handler)

	// Preflight should return 204 No Content
	Expect(status).Equals(http.StatusNoContent)
	Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals("https://app.example.com")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).ContainsSubstring("POST")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).ContainsSubstring("GET")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).ContainsSubstring("PUT")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).ContainsSubstring("PATCH")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).ContainsSubstring("DELETE")
	Expect(response.Header().Get("Access-Control-Allow-Methods")).ContainsSubstring("OPTIONS")
	Expect(response.Header().Get("Access-Control-Allow-Headers")).ContainsSubstring("Authorization")
	Expect(response.Header().Get("Access-Control-Allow-Headers")).ContainsSubstring("Content-Type")
	Expect(response.Header().Get("Access-Control-Allow-Headers")).ContainsSubstring("X-Tenant-ID")
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("true")
	Expect(response.Header().Get("Access-Control-Max-Age")).Equals("600")
}

func TestCORS_AllMethodsSupported(t *testing.T) {
	RegisterT(t)

	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	os.Setenv("ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	defer func() {
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("ALLOWED_METHODS")
	}()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	server.AddHeader("Origin", "https://app.example.com")
	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	allowedMethods := response.Header().Get("Access-Control-Allow-Methods")
	
	// Verify all required methods are present
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		Expect(strings.Contains(allowedMethods, method)).IsTrue()
	}
}

func TestCORS_RequiredHeadersSupported(t *testing.T) {
	RegisterT(t)

	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	os.Setenv("ALLOWED_HEADERS", "Authorization,Content-Type,X-Tenant-ID")
	defer func() {
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("ALLOWED_HEADERS")
	}()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	server.AddHeader("Origin", "https://app.example.com")
	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	allowedHeaders := response.Header().Get("Access-Control-Allow-Headers")
	
	// Verify all required headers are present
	Expect(strings.Contains(allowedHeaders, "Authorization")).IsTrue()
	Expect(strings.Contains(allowedHeaders, "Content-Type")).IsTrue()
	Expect(strings.Contains(allowedHeaders, "X-Tenant-ID")).IsTrue()
}

func TestCORS_CredentialsEnabled(t *testing.T) {
	RegisterT(t)

	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	os.Setenv("CORS_ALLOW_CREDENTIALS", "true")
	defer func() {
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("CORS_ALLOW_CREDENTIALS")
	}()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	server.AddHeader("Origin", "https://app.example.com")
	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("true")
}

func TestCORS_PreflightCachingMaxAge(t *testing.T) {
	RegisterT(t)

	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	server.AddHeader("Origin", "https://app.example.com")
	server.AddHeader("Access-Control-Request-Method", "POST")
	
	status, response := server.ExecuteAsOptions(handler)

	Expect(status).Equals(http.StatusNoContent)
	Expect(response.Header().Get("Access-Control-Max-Age")).Equals("600")
}

func TestCORS_WildcardOriginRejectedInProduction(t *testing.T) {
	RegisterT(t)

	// Simulate production environment by not setting ALLOWED_ORIGINS or setting it to wildcard
	os.Setenv("ALLOWED_ORIGINS", "*")
	os.Setenv("GO_ENV", "production")
	defer func() {
		os.Unsetenv("ALLOWED_ORIGINS")
		os.Unsetenv("GO_ENV")
	}()

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	server.AddHeader("Origin", "https://random.com")
	status, response := server.Execute(handler)

	Expect(status).Equals(http.StatusOK)
	// In production with wildcard configured, should not allow arbitrary origins
	// The middleware should reject or require explicit origin configuration
	origin := response.Header().Get("Access-Control-Allow-Origin")
	Expect(origin).NotEquals("*")
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	RegisterT(t)

	// Configure multiple allowed origins
	os.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://staging.app.example.com,https://dev.app.example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	server := mock.NewServer()
	server.Use(middlewares.CORS())
	handler := func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	}

	// Test each allowed origin
	testOrigins := []string{
		"https://app.example.com",
		"https://staging.app.example.com",
		"https://dev.app.example.com",
	}

	for _, origin := range testOrigins {
		server = mock.NewServer()
		server.Use(middlewares.CORS())
		server.AddHeader("Origin", origin)
		status, response := server.Execute(handler)

		Expect(status).Equals(http.StatusOK)
		Expect(response.Header().Get("Access-Control-Allow-Origin")).Equals(origin)
		Expect(response.Header().Get("Access-Control-Allow-Credentials")).Equals("true")
	}
}
