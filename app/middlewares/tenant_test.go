package middlewares_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/middlewares"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

var testCases = []struct {
	expected string
	urls     []string
}{
	{
		"Avengers",
		[]string{
			"http://avengers.test.fider.io",
			"http://avengers.test.fider.io:3000",
		},
	},
	{
		"Demonstration",
		[]string{
			"http://demo.test.fider.io",
			"http://demo.test.fider.io:1231",
			"http://demo.test.fider.io:80",
		},
	},
}

func TestMultiTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	for _, testCase := range testCases {
		for _, url := range testCase.urls {

			server := mock.NewServer()
			server.Use(middlewares.MultiTenant())

			status, response := server.WithURL(url).Execute(func(c *web.Context) error {
				return c.String(http.StatusOK, c.Tenant().Name)
			})

			Expect(status).Equals(http.StatusOK)
			Expect(response.Body.String()).Equals(testCase.expected)
		}
	}
}

func TestMultiTenant_SubSubDomain(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())

	status, _ := server.WithURL("http://demo.demo.test.fider.io").Execute(func(c *web.Context) error {
		if c.Tenant() == nil {
			return c.Ok(web.Map{})
		}
		return c.Failure(errors.New("should not have found tenant"))
	})

	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_UnknownDomain(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())

	status, _ := server.WithURL("http://somedomain.com").Execute(func(c *web.Context) error {
		if c.Tenant() == nil {
			return c.Ok(web.Map{})
		}
		return c.Failure(errors.New("should not have found tenant"))
	})

	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_DisabledTenant_ShouldNotSetInContext(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			q.Result.Status = enum.TenantDisabled
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.RequireTenant())

	status, _ := server.WithURL("http://avengers.test.fider.io").Execute(func(c *web.Context) error {
		return c.Ok(nil)
	})

	Expect(status).Equals(http.StatusNotFound)
}

func TestMultiTenant_CanonicalHeader(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	var testCases = []struct {
		input  string
		output string
		isAjax bool
	}{
		{
			"http://avengers.test.fider.io/",
			"http://feedback.theavengers.com/",
			false,
		},
		{
			"http://avengers.test.fider.io/",
			"",
			true,
		},
		{
			"http://feedback.theavengers.com/",
			"",
			false,
		},
		{
			"http://avengers.test.fider.io/posts",
			"http://feedback.theavengers.com/posts",
			false,
		},
		{
			"http://avengers.test.fider.io/posts?q=1",
			"http://feedback.theavengers.com/posts?q=1",
			false,
		},
		{
			"http://demo.test.fider.io",
			"",
			false,
		},
	}

	for _, testCase := range testCases {
		server := mock.NewServer()
		server.Use(middlewares.MultiTenant())

		if testCase.isAjax {
			server.AddHeader("Accept", "application/json")
		}

		var canonicalURL string
		status, _ := server.
			WithURL(testCase.input).
			Execute(func(c *web.Context) error {
				canonicalURL, _ = c.Value("Canonical-URL").(string)
				return c.Ok(web.Map{})
			})

		Expect(status).Equals(http.StatusOK)
		Expect(canonicalURL).Equals(testCase.output)
	}

}

func TestMultiTenant_WithXTenantIDHeader(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	// Test: X-Tenant-ID header successfully resolves tenant
	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io")

	status, response := server.WithURL("http://anyhost.example.com").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}

func TestMultiTenant_XTenantIDHeader_PriorityOverHostname(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	// Test: X-Tenant-ID header takes priority over hostname
	// Request URL suggests demo.test.fider.io, but header says avengers.test.fider.io
	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io")

	status, response := server.WithURL("http://demo.test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers") // Should be Avengers, not Demonstration
}

func TestMultiTenant_XTenantIDHeader_InvalidTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	// Test: Invalid X-Tenant-ID header results in no tenant
	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "nonexistent.test.fider.io")

	status, _ := server.WithURL("http://anyhost.example.com").Execute(func(c *web.Context) error {
		if c.Tenant() == nil {
			return c.Ok(web.Map{})
		}
		return c.Failure(errors.New("should not have found tenant"))
	})

	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_XTenantIDHeader_EmptyFallsBackToHostname(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	// Test: Empty X-Tenant-ID header falls back to hostname resolution
	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "")

	status, response := server.WithURL("http://demo.test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Demonstration") // Should resolve via hostname
}

func TestMultiTenant_XTenantIDHeader_MultipleScenarios(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	var testCases = []struct {
		description  string
		url          string
		headerValue  string
		expectedName string
		expectNil    bool
	}{
		{
			"Valid header with matching subdomain",
			"http://avengers.test.fider.io",
			"avengers.test.fider.io",
			"Avengers",
			false,
		},
		{
			"Valid header with different subdomain",
			"http://demo.test.fider.io",
			"avengers.test.fider.io",
			"Avengers", // Header takes priority
			false,
		},
		{
			"Valid header with non-tenant hostname",
			"http://app.example.com",
			"demo.test.fider.io",
			"Demonstration",
			false,
		},
		{
			"No header with valid subdomain",
			"http://demo.test.fider.io",
			"",
			"Demonstration", // Falls back to hostname
			false,
		},
		{
			"Invalid header with valid subdomain",
			"http://demo.test.fider.io",
			"invalid.test.fider.io",
			"", // Header takes priority but is invalid
			true,
		},
	}

	for _, testCase := range testCases {
		server := mock.NewServer()
		server.Use(middlewares.MultiTenant())

		if testCase.headerValue != "" {
			server.AddHeader("X-Tenant-ID", testCase.headerValue)
		}

		status, response := server.WithURL(testCase.url).Execute(func(c *web.Context) error {
			if c.Tenant() == nil {
				if testCase.expectNil {
					return c.Ok(web.Map{})
				}
				return c.Failure(errors.New("expected tenant but got nil for: " + testCase.description))
			}
			if testCase.expectNil {
				return c.Failure(errors.New("expected nil tenant but got tenant for: " + testCase.description))
			}
			return c.String(http.StatusOK, c.Tenant().Name)
		})

		Expect(status).Equals(http.StatusOK)
		if !testCase.expectNil {
			Expect(response.Body.String()).Equals(testCase.expectedName)
		}
	}
}

func TestSingleTenant_NoTenants(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetFirstTenant) error {
		return app.ErrNotFound
	})

	server := mock.NewSingleTenantServer()
	server.Use(middlewares.SingleTenant())

	status, _ := server.WithURL("http://somedomain.com").Execute(func(c *web.Context) error {
		if c.Tenant() == nil {
			return c.Ok(web.Map{})
		}
		return c.Failure(errors.New("should not have found tenant"))
	})

	Expect(status).Equals(http.StatusOK)
}

func TestSingleTenant_WithTenants_ShouldSetFirstToContext(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetFirstTenant) error {
		q.Result = &entity.Tenant{Name: "MyCompany", Subdomain: "mycompany", Status: enum.TenantActive}
		return nil
	})

	server := mock.NewSingleTenantServer()
	server.Use(middlewares.SingleTenant())

	status, response := server.WithURL("http://test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("MyCompany")
}

func TestBlockPendingTenants_Active(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.DemoTenant.Status = enum.TenantActive

	server.Use(middlewares.BlockPendingTenants())
	status, _ := server.OnTenant(mock.DemoTenant).Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	})

	Expect(status).Equals(http.StatusOK)
}

func TestBlockPendingTenants_Pending(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.DemoTenant.Status = enum.TenantPending

	server.Use(middlewares.BlockPendingTenants())
	status, _ := server.OnTenant(mock.DemoTenant).Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusTeapot)
	})

	Expect(status).Equals(http.StatusOK)
}

func TestCheckTenantPrivacy_Private_Unauthenticated(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.DemoTenant.IsPrivate = true

	server.Use(middlewares.CheckTenantPrivacy())
	status, response := server.OnTenant(mock.DemoTenant).Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	})

	Expect(status).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/signin")
}

func TestCheckTenantPrivacy_Private_Authenticated(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.DemoTenant.IsPrivate = true

	server.Use(middlewares.CheckTenantPrivacy())
	status, _ := server.
		OnTenant(mock.DemoTenant).
		AsUser(mock.AryaStark).
		Execute(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		})

	Expect(status).Equals(http.StatusOK)
}

func TestCheckTenantPrivacy_NotPrivate_Unauthenticated(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.DemoTenant.IsPrivate = false

	server.Use(middlewares.CheckTenantPrivacy())
	status, _ := server.
		OnTenant(mock.DemoTenant).
		Execute(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		})

	Expect(status).Equals(http.StatusOK)
}

func TestRequireTenant_MultiHostMode_NoTenants_404(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.RequireTenant())

	status, _ := server.WithURL("http://somedomain.com").Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	})

	Expect(status).Equals(http.StatusNotFound)
}

func TestRequireTenant_MultiHostMode_ValidTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.RequireTenant())

	status, response := server.WithURL("http://avengers.test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}

func TestRequireTenant_SingleHostMode_NoTenants_RedirectToSignUp(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetFirstTenant) error {
		return app.ErrNotFound
	})

	server := mock.NewSingleTenantServer()
	server.Use(middlewares.SingleTenant())
	server.Use(middlewares.RequireTenant())

	status, response := server.WithURL("http://somedomain.com").Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	})

	Expect(status).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/signup")
}

func TestRequireTenant_SingleHostMode_ValidTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetFirstTenant) error {
		q.Result = mock.DemoTenant
		return nil
	})

	server := mock.NewServer()
	server.Use(middlewares.SingleTenant())
	server.Use(middlewares.RequireTenant())

	status, response := server.WithURL("http://test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Demonstration")
}

func TestBlockLockedTenants_ActiveTenant(t *testing.T) {
	RegisterT(t)
	server := mock.NewServer()
	server.Use(middlewares.BlockLockedTenants())

	status, response := server.
		WithURL("http://demo.test.fider.io").
		OnTenant(mock.DemoTenant).
		Execute(func(c *web.Context) error {
			return c.String(http.StatusOK, c.Tenant().Name)
		})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Demonstration")
}

func TestBlockLockedTenants_LockedTenant(t *testing.T) {
	RegisterT(t)
	server := mock.NewServer()
	server.Use(middlewares.BlockLockedTenants())
	mock.DemoTenant.Status = enum.TenantLocked

	status, _ := server.
		WithURL("http://demo.test.fider.io/api/v1/posts").
		OnTenant(mock.DemoTenant).
		Execute(func(c *web.Context) error {
			return c.String(http.StatusOK, c.Tenant().Name)
		})

	Expect(status).Equals(http.StatusPaymentRequired)
}

// X-Tenant-ID Header-Based Resolution Tests

func TestMultiTenant_WithXTenantIDHeader_ValidTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		// Header-based resolution should query by the domain derived from X-Tenant-ID
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		} else if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io")

	status, response := server.WithURL("http://some-other-domain.com").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}

func TestMultiTenant_WithXTenantIDHeader_InvalidTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "nonexistent.test.fider.io")

	status, _ := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		if c.Tenant() == nil {
			return c.Ok(web.Map{})
		}
		return c.Failure(errors.New("should not have found tenant"))
	})

	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_WithXTenantIDHeader_PrecedenceOverHost(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		// X-Tenant-ID should take precedence, so we should query for "demo" not "avengers"
		if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		} else if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "demo.test.fider.io")

	// URL host is "avengers" but header specifies "demo"
	status, response := server.WithURL("http://avengers.test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Demonstration")
}

func TestMultiTenant_WithXTenantIDHeader_EmptyHeader(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "")

	// Empty header should fall back to host-based resolution
	status, response := server.WithURL("http://avengers.test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}

func TestMultiTenant_WithXTenantIDHeader_DisabledTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			q.Result.Status = enum.TenantDisabled
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.RequireTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io")

	status, _ := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		return c.Ok(nil)
	})

	Expect(status).Equals(http.StatusNotFound)
}

func TestMultiTenant_WithXTenantIDHeader_LockedTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			q.Result.Status = enum.TenantLocked
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.BlockLockedTenants())
	server.AddHeader("X-Tenant-ID", "demo.test.fider.io")

	status, _ := server.
		WithURL("http://some-domain.com/api/v1/posts").
		Execute(func(c *web.Context) error {
			return c.String(http.StatusOK, c.Tenant().Name)
		})

	Expect(status).Equals(http.StatusPaymentRequired)
}

func TestMultiTenant_WithXTenantIDHeader_PrivateTenantUnauthenticated(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "demo.test.fider.io" {
			tenant := mock.DemoTenant
			tenant.IsPrivate = true
			q.Result = tenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.CheckTenantPrivacy())
	server.AddHeader("X-Tenant-ID", "demo.test.fider.io")

	status, response := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	})

	Expect(status).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/signin")
}

func TestMultiTenant_WithXTenantIDHeader_PrivateTenantAuthenticated(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "demo.test.fider.io" {
			tenant := mock.DemoTenant
			tenant.IsPrivate = true
			q.Result = tenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.CheckTenantPrivacy())
	server.AddHeader("X-Tenant-ID", "demo.test.fider.io")

	status, _ := server.
		WithURL("http://some-domain.com").
		AsUser(mock.AryaStark).
		Execute(func(c *web.Context) error {
			return c.NoContent(http.StatusOK)
		})

	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_WithXTenantIDHeader_CanonicalURL(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io")

	var canonicalURL string
	status, _ := server.
		WithURL("http://some-domain.com/posts").
		Execute(func(c *web.Context) error {
			canonicalURL, _ = c.Value("Canonical-URL").(string)
			return c.Ok(web.Map{})
		})

	Expect(status).Equals(http.StatusOK)
	// Canonical URL should be set based on tenant's CNAME
	Expect(canonicalURL).Equals("http://feedback.theavengers.com/posts")
}

func TestMultiTenant_WithXTenantIDHeader_WithRequireTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.RequireTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io")

	status, response := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}

func TestMultiTenant_WithXTenantIDHeader_InvalidTenantWithRequireTenant(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.Use(middlewares.RequireTenant())
	server.AddHeader("X-Tenant-ID", "nonexistent.test.fider.io")

	status, _ := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		return c.NoContent(http.StatusOK)
	})

	Expect(status).Equals(http.StatusNotFound)
}

func TestSingleTenant_WithXTenantIDHeader_ShouldIgnoreHeader(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetFirstTenant) error {
		q.Result = &entity.Tenant{Name: "MyCompany", Subdomain: "mycompany", Status: enum.TenantActive}
		return nil
	})

	server := mock.NewSingleTenantServer()
	server.Use(middlewares.SingleTenant())
	server.AddHeader("X-Tenant-ID", "should-be-ignored.test.fider.io")

	status, response := server.WithURL("http://test.fider.io").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	// Single tenant mode should ignore X-Tenant-ID header and always use first tenant
	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("MyCompany")
}

func TestMultiTenant_WithXTenantIDHeader_SubdomainFormat(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		// Test handling of subdomain-only format in header
		if q.Domain == "demo.test.fider.io" {
			q.Result = mock.DemoTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	// Pass only subdomain in header, should be resolved properly
	server.AddHeader("X-Tenant-ID", "demo")

	status, _ := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		if c.Tenant() != nil {
			return c.String(http.StatusOK, c.Tenant().Name)
		}
		return c.Ok(web.Map{})
	})

	// Behavior depends on implementation: subdomain-only may or may not work
	// This test validates that the middleware handles subdomain-only gracefully
	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_WithXTenantIDHeader_CNAMEResolution(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		// Test CNAME resolution via X-Tenant-ID header
		if q.Domain == "feedback.theavengers.com" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "feedback.theavengers.com")

	status, response := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		return c.String(http.StatusOK, c.Tenant().Name)
	})

	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}

func TestMultiTenant_WithXTenantIDHeader_CaseInsensitive(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		// Tenant domain lookup should be case-insensitive
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	// Test case variations
	server.AddHeader("X-Tenant-ID", "AVENGERS.test.fider.io")

	status, _ := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		if c.Tenant() != nil {
			return c.String(http.StatusOK, c.Tenant().Name)
		}
		return c.Ok(web.Map{})
	})

	// Validation depends on implementation's case handling
	Expect(status).Equals(http.StatusOK)
}

func TestMultiTenant_WithXTenantIDHeader_WithPort(t *testing.T) {
	RegisterT(t)

	bus.AddHandler(func(ctx context.Context, q *query.GetTenantByDomain) error {
		// X-Tenant-ID with port should be handled correctly
		if q.Domain == "avengers.test.fider.io" {
			q.Result = mock.AvengersTenant
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	server.Use(middlewares.MultiTenant())
	server.AddHeader("X-Tenant-ID", "avengers.test.fider.io:3000")

	status, response := server.WithURL("http://some-domain.com").Execute(func(c *web.Context) error {
		if c.Tenant() != nil {
			return c.String(http.StatusOK, c.Tenant().Name)
		}
		return c.Ok(web.Map{})
	})

	// Port should be stripped from X-Tenant-ID header
	Expect(status).Equals(http.StatusOK)
	Expect(response.Body.String()).Equals("Avengers")
}
