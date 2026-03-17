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

func TestCORS(t *testing.T) {
	RegisterT(t)

	env.Config.AllowedOrigins = "http://localhost:3001"
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
