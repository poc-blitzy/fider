package web_test

import (
	"crypto/tls"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/getfider/fider/app/models/entity"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/web"
)

// testPort returns the port suffix from env.Config.Port for constructing test URLs
var testPort = ":" + env.Config.Port

func newGetContext(rawurl string, headers map[string]string) *web.Context {
	u, _ := url.Parse(rawurl)
	e := web.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", u.RequestURI(), nil)
	req.Host = u.Host

	if u.Scheme == "https" {
		req.TLS = &tls.ConnectionState{}
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return web.NewContext(e, req, res, nil)
}

func newBodyContext(method string, params web.StringMap, body, contentType string) *web.Context {
	e := web.New()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/some/resource", strings.NewReader(body))
	req.Host = "demo.test.fider.io" + testPort
	req.Header.Set("Content-Type", contentType)
	return web.NewContext(e, req, res, params)
}

func TestContextID(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://demo.test.fider.io"+testPort, nil)

	Expect(ctx.ContextID()).IsNotEmpty()
	Expect(ctx.ContextID()).HasLen(32)
}

func TestBaseURL(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://demo.test.fider.io"+testPort, nil)

	Expect(ctx.BaseURL()).Equals("http://demo.test.fider.io"+testPort)
}

func TestBaseURL_HTTPS(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("https://demo.test.fider.io"+testPort, nil)

	Expect(ctx.BaseURL()).Equals("https://demo.test.fider.io"+testPort)
}

func TestBaseURL_HTTPS_Proxy(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://demo.test.fider.io"+testPort, map[string]string{
		"X-Forwarded-Proto": "https",
	})

	Expect(ctx.BaseURL()).Equals(fmt.Sprintf("https://demo.test.fider.io%s", testPort))
}

func TestCurrentURL(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://demo.test.fider.io"+testPort+"/resource?id=23", nil)

	Expect(ctx.Request.URL.String()).Equals(fmt.Sprintf("http://demo.test.fider.io%s/resource?id=23", testPort))
}

func TestTenantURL(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://login.test.fider.io"+testPort, nil)
	tenant := &entity.Tenant{
		ID:        1,
		Subdomain: "theavengers",
	}
	Expect(web.TenantBaseURL(ctx, tenant)).Equals("http://theavengers.test.fider.io"+testPort)
}

func TestTenantURL_WithCNAME(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://demo.test.fider.io"+testPort, nil)
	tenant := &entity.Tenant{
		ID:        1,
		Subdomain: "theavengers",
		CNAME:     "feedback.theavengers.com",
	}
	Expect(web.TenantBaseURL(ctx, tenant)).Equals("http://feedback.theavengers.com"+testPort)
}

func TestTenantURL_SingleHostMode(t *testing.T) {
	RegisterT(t)
	env.Config.HostMode = "single"

	ctx := newGetContext("http://demo.test.fider.io"+testPort, nil)
	tenant := &entity.Tenant{
		ID:        1,
		Subdomain: "theavengers",
	}
	Expect(web.TenantBaseURL(ctx, tenant)).Equals("https://test.fider.io"+testPort)
}

func TestAssetsURL_SingleHostMode(t *testing.T) {
	RegisterT(t)

	env.Config.HostMode = "single"
	ctx := newGetContext("http://feedback.theavengers.com"+testPort, nil)
	ctx.SetTenant(&entity.Tenant{
		ID:        1,
		Subdomain: "theavengers",
	})

	Expect(web.AssetsURL(ctx, "/assets/main.js")).Equals(fmt.Sprintf("https://test.fider.io%s/assets/main.js", testPort))
	Expect(web.AssetsURL(ctx, "/assets/main.css")).Equals(fmt.Sprintf("https://test.fider.io%s/assets/main.css", testPort))

	env.Config.CDN.Host = "fidercdn.com"
	Expect(web.AssetsURL(ctx, "/assets/main.js")).Equals("http://fidercdn.com/assets/main.js")
	Expect(web.AssetsURL(ctx, "/assets/main.css")).Equals("http://fidercdn.com/assets/main.css")
}

func TestAssetsURL_MultiHostMode(t *testing.T) {
	RegisterT(t)

	env.Config.HostMode = "multi"
	ctx := newGetContext("http://theavengers.test.fider.io"+testPort, nil)
	ctx.SetTenant(&entity.Tenant{
		ID:        1,
		Subdomain: "theavengers",
		CNAME:     "feedback.theavengers.com",
	})

	Expect(web.AssetsURL(ctx, "/assets/main.js")).Equals("http://theavengers.test.fider.io" + testPort + "/assets/main.js")
	Expect(web.AssetsURL(ctx, "/assets/main.css")).Equals("http://theavengers.test.fider.io" + testPort + "/assets/main.css")

	env.Config.CDN.Host = "fidercdn.com"
	Expect(web.AssetsURL(ctx, "/assets/main.js")).Equals("http://theavengers.fidercdn.com/assets/main.js")
	Expect(web.AssetsURL(ctx, "/assets/main.css")).Equals("http://theavengers.fidercdn.com/assets/main.css")
}

func TestCanonicalURL_SameDomain(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://theavengers.test.fider.io"+testPort, nil)

	ctx.SetCanonicalURL("")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://theavengers.test.fider.io` + testPort)

	ctx.SetCanonicalURL("/some-url")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://theavengers.test.fider.io` + testPort + `/some-url`)

	ctx.SetCanonicalURL("/some-other-url")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://theavengers.test.fider.io` + testPort + `/some-other-url`)

	ctx.SetCanonicalURL("page-b/abc.html")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://theavengers.test.fider.io` + testPort + `/page-b/abc.html`)
}

func TestCanonicalURL_DifferentDomain(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://theavengers.test.fider.io"+testPort, nil)

	ctx.SetCanonicalURL("http://feedback.theavengers.com/some-url")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://feedback.theavengers.com/some-url`)

	ctx.SetCanonicalURL("")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://feedback.theavengers.com`)

	ctx.SetCanonicalURL("/some-other-url")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://feedback.theavengers.com/some-other-url`)

	ctx.SetCanonicalURL("page-b/abc.html")
	Expect(ctx.Value("Canonical-URL")).Equals(`http://feedback.theavengers.com/page-b/abc.html`)
}

func TestGetOAuthBaseURL(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("https://mydomain.com/hello-world", nil)

	env.Config.HostMode = "multi"
	Expect(web.OAuthBaseURL(ctx)).Equals("https://login.test.fider.io")

	env.Config.HostMode = "single"
	Expect(web.OAuthBaseURL(ctx)).Equals("https://test.fider.io" + testPort)
}

func TestGetOAuthBaseURL_WithPort(t *testing.T) {
	RegisterT(t)

	ctx := newGetContext("http://demo.test.fider.io" + testPort + "/hello-world", nil)

	env.Config.HostMode = "multi"
	Expect(web.OAuthBaseURL(ctx)).Equals("http://login.test.fider.io" + testPort)

	env.Config.HostMode = "single"
	Expect(web.OAuthBaseURL(ctx)).Equals("https://test.fider.io" + testPort)
}
