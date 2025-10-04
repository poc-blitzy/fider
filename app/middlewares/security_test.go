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

func TestSecureWithoutCDN(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	var ctxID string
	status, response := server.Execute(func(c *web.Context) error {
		ctxID = c.ContextID()
		return c.NoContent(http.StatusOK)
	})

	expectedPolicy := "base-uri 'self'; default-src 'self'; style-src 'self' 'unsafe-inline' https://*.paddle.com ; script-src 'self' 'nonce-" + ctxID + "' https://www.google-analytics.com https://*.paddle.com ; img-src 'self' https: data: ; font-src 'self' data: ; object-src 'none'; media-src 'none'; connect-src 'self' https://www.google-analytics.com ; frame-src 'self' https://*.paddle.com"

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Content-Security-Policy")).Equals(expectedPolicy)
	Expect(response.Header().Get("X-XSS-Protection")).Equals("1; mode=block")
	Expect(response.Header().Get("X-Content-Type-Options")).Equals("nosniff")
	Expect(response.Header().Get("Referrer-Policy")).Equals("no-referrer-when-downgrade")
}

func TestSecureWithCDN(t *testing.T) {
	RegisterT(t)

	env.Config.CDN.Host = "test.fider.io"

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	var ctxID string
	status, response := server.Execute(func(c *web.Context) error {
		ctxID = c.ContextID()
		return c.NoContent(http.StatusOK)
	})

	expectedPolicy := "base-uri 'self'; default-src 'self'; style-src 'self' 'unsafe-inline' https://*.paddle.com *.test.fider.io; script-src 'self' 'nonce-" + ctxID + "' https://www.google-analytics.com https://*.paddle.com *.test.fider.io; img-src 'self' https: data: *.test.fider.io; font-src 'self' data: *.test.fider.io; object-src 'none'; media-src 'none'; connect-src 'self' https://www.google-analytics.com *.test.fider.io; frame-src 'self' https://*.paddle.com"

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Content-Security-Policy")).Equals(expectedPolicy)
	Expect(response.Header().Get("X-XSS-Protection")).Equals("1; mode=block")
	Expect(response.Header().Get("X-Content-Type-Options")).Equals("nosniff")
	Expect(response.Header().Get("Referrer-Policy")).Equals("no-referrer-when-downgrade")
}

func TestSecureWithCDN_SingleHost(t *testing.T) {
	RegisterT(t)

	env.Config.CDN.Host = "test.fider.io"

	server := mock.NewSingleTenantServer()
	server.Use(middlewares.Secure())

	var ctxID string
	status, response := server.WithURL("http://test.fider.io").Execute(func(c *web.Context) error {
		ctxID = c.ContextID()
		return c.NoContent(http.StatusOK)
	})

	expectedPolicy := "base-uri 'self'; default-src 'self'; style-src 'self' 'unsafe-inline' https://*.paddle.com test.fider.io; script-src 'self' 'nonce-" + ctxID + "' https://www.google-analytics.com https://*.paddle.com test.fider.io; img-src 'self' https: data: test.fider.io; font-src 'self' data: test.fider.io; object-src 'none'; media-src 'none'; connect-src 'self' https://www.google-analytics.com test.fider.io; frame-src 'self' https://*.paddle.com"

	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Content-Security-Policy")).Equals(expectedPolicy)
	Expect(response.Header().Get("X-XSS-Protection")).Equals("1; mode=block")
	Expect(response.Header().Get("X-Content-Type-Options")).Equals("nosniff")
	Expect(response.Header().Get("Referrer-Policy")).Equals("no-referrer-when-downgrade")
}

func TestCSRF_ReadRequest_NoCSRFCheck(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	status, _ := server.
		WithURL("http://example.com").
		OnTenant(mock.DemoTenant).
		Execute(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		})

	Expect(status).Equals(http.StatusOK)
}

func TestCSRF_WriteRequest_NoCSRFToken_Returns403(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.CSRF())

	// Send form data (not JSON) to test CSRF protection
	// JSON requests are exempt from CSRF as they are AJAX requests protected by CORS
	status, _ := server.
		WithURL("http://example.com").
		OnTenant(mock.DemoTenant).
		ExecutePostForm(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		}, map[string]string{"test": "data"})

	Expect(status).Equals(http.StatusForbidden)
}

func TestCSRF_WriteRequest_WithBearerToken_NoCSRFCheck(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	status, _ := server.
		WithURL("http://example.com").
		OnTenant(mock.DemoTenant).
		AddHeader("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token").
		ExecutePost(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		}, `{"test":"data"}`)

	Expect(status).Equals(http.StatusOK)
}

func TestCSRF_WriteRequest_AjaxRequest_NoCSRFCheck(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	status, _ := server.
		WithURL("http://example.com").
		OnTenant(mock.DemoTenant).
		AddHeader("X-Requested-With", "XMLHttpRequest").
		ExecutePost(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		}, `{"test":"data"}`)

	Expect(status).Equals(http.StatusOK)
}

func TestCSRF_DeleteRequest_WithBearerToken_NoCSRFCheck(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	status, _ := server.
		WithURL("http://example.com").
		OnTenant(mock.DemoTenant).
		AddHeader("Authorization", "Bearer valid.jwt.token").
		ExecuteDelete(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		})

	Expect(status).Equals(http.StatusOK)
}

func TestCSRF_PutRequest_WithBearerToken_NoCSRFCheck(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Secure())

	status, _ := server.
		WithURL("http://example.com").
		OnTenant(mock.DemoTenant).
		AddHeader("Authorization", "Bearer valid.jwt.token").
		ExecutePut(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		}, `{"test":"data"}`)

	Expect(status).Equals(http.StatusOK)
}
