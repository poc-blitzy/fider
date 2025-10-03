package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/jsonq"
	"github.com/getfider/fider/app/pkg/web"
	"github.com/julienschmidt/httprouter"
)

// Server is a HTTP server wrapper for testing purpose
type Server struct {
	engine      *web.Engine
	context     *web.Context
	recorder    *httptest.ResponseRecorder
	middleware  []web.MiddlewareFunc
	params      map[string]string  // Store parameters to preserve them across context resets
	httpRequest *http.Request      // Store original http.Request for context recreation
}

func createServer() *Server {
	bus.AddHandler(func(ctx context.Context, q *query.ListActiveOAuthProviders) error {
		return nil
	})

	engine := web.New()

	// Create a new request and set matched routed into context
	request, _ := http.NewRequest("GET", "/", nil)
	request = request.WithContext(context.WithValue(request.Context(), httprouter.ParamsKey, httprouter.Params{
		httprouter.Param{Key: httprouter.MatchedRoutePathParam, Value: "/"},
	}))

	recorder := httptest.NewRecorder()
	params := make(web.StringMap)
	context := web.NewContext(engine, request, recorder, params)

	return &Server{
		engine:      engine,
		recorder:    recorder,
		context:     context,
		middleware:  []web.MiddlewareFunc{},
		params:      make(map[string]string), // Initialize params map for test parameter preservation
		httpRequest: request,                  // Store original http.Request for context recreation
	}
}

// Engine returns current engine from mocked server
func (s *Server) Engine() *web.Engine {
	return s.engine
}

// Use adds a new middleware to pipeline
func (s *Server) Use(middleware web.MiddlewareFunc) *Server {
	if middleware != nil {
		s.middleware = append(s.middleware, middleware)
	}
	return s
}

// OnTenant set current context tenant
func (s *Server) OnTenant(tenant *entity.Tenant) *Server {
	s.context.SetTenant(tenant)
	return s
}

// AsUser set current context user
func (s *Server) AsUser(user *entity.User) *Server {
	s.context.SetUser(user)
	return s
}

// AddParam to current context route parameters
func (s *Server) AddParam(name string, value any) *Server {
	strValue := fmt.Sprintf("%v", value)
	s.params[name] = strValue // Save parameter for later restoration
	s.context.AddParam(name, strValue)
	return s
}

// AddHeader add key-value to current context headers
func (s *Server) AddHeader(name string, value string) *Server {
	s.context.Request.SetHeader(name, value)
	return s
}

// AddCookie add key-value to current context cookies
func (s *Server) AddCookie(name string, value string) *Server {
	s.context.Request.AddCookie(&http.Cookie{Name: name, Value: value})
	return s
}

// WithURL set current context Request URL
func (s *Server) WithURL(fullURL string) *Server {
	u, _ := url.Parse(fullURL)
	s.context.Request.URL = u
	return s
}

// Execute given handler and return response
func (s *Server) Execute(handler web.HandlerFunc) (int, *httptest.ResponseRecorder) {
	// Reset the recorder for each execution to prevent state carryover
	s.recorder = httptest.NewRecorder()
	
	// Preserve tenant and user from old context before creating new one
	oldTenant := s.context.Tenant()
	oldUser := s.context.User()
	
	// Pass saved parameters to new context (no need to restore individually since they're passed to NewContext)
	s.context = web.NewContext(s.engine, s.httpRequest, s.recorder, s.params)
	
	// Restore tenant and user to the new context
	if oldTenant != nil {
		s.context.SetTenant(oldTenant)
	}
	if oldUser != nil {
		s.context.SetUser(oldUser)
	}
	
	next := handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		next = s.middleware[i](next)
	}

	if err := next(s.context); err != nil {
		// Only call Failure if handler hasn't already written a response
		// If Response.StatusCode != 0, handler already called WriteHeader (e.g., via c.JSON, c.Blob)
		// and we should not overwrite the response
		if s.context.Response.StatusCode == 0 {
			// Handler errored without writing a response, so write error page
			if failureErr := s.context.Failure(err); failureErr != nil {
				panic(fmt.Sprintf("failed to write error response: %v (original error: %v)", failureErr, err))
			}
		}
		// If StatusCode != 0, handler already wrote response (possibly with error status)
		// Don't overwrite it by calling Failure
	}
	return s.recorder.Code, s.recorder
}

// ExecuteAsJSON given handler and return json response
func (s *Server) ExecuteAsJSON(handler web.HandlerFunc) (int, *jsonq.Query) {
	code, response := s.Execute(handler)
	return code, toJSONQuery(response)
}

// ExecutePost executes given handler as POST and return response
func (s *Server) ExecutePost(handler web.HandlerFunc, body string) (int, *httptest.ResponseRecorder) {
	// Modify the underlying http.Request BEFORE calling Execute
	// This ensures the modifications persist when Execute creates a new context
	s.httpRequest.Method = "POST"
	s.httpRequest.Body = io.NopCloser(strings.NewReader(body))
	s.httpRequest.ContentLength = int64(len(body))
	s.httpRequest.Header.Set("Content-Type", web.UTF8JSONContentType)

	return s.Execute(handler)
}

// ExecuteAsOptions executes given handler as OPTIONS request and return response
func (s *Server) ExecuteAsOptions(handler web.HandlerFunc) (int, *httptest.ResponseRecorder) {
	s.context.Request.Method = "OPTIONS"
	return s.Execute(handler)
}

// ExecutePostAsJSON executes given handler as POST and return json response
func (s *Server) ExecutePostAsJSON(handler web.HandlerFunc, body string) (int, *jsonq.Query) {
	code, response := s.ExecutePost(handler, body)
	return code, toJSONQuery(response)
}

// ExecuteAsPage given handler and return page props
func (s *Server) ExecuteAsPage(handler web.HandlerFunc) (int, *web.Props) {
	code, response := s.Execute(handler)
	bodyString := response.Body.String()

	startTag := "<script id=\"server-data\" type=\"application/json\">"
	endTag := "</script>"

	startIndex := strings.Index(bodyString, startTag) + len(startTag)
	endIndex := strings.Index(bodyString[startIndex:], endTag)

	serverData := strings.TrimSpace(bodyString[startIndex : startIndex+endIndex])
	serverDataJSON := map[string]any{}

	err := json.Unmarshal([]byte(serverData), &serverDataJSON)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse server data"))
	}

	title, _ := serverDataJSON["title"].(string)
	description, _ := serverDataJSON["description"].(string)
	page, _ := serverDataJSON["page"].(string)
	props, _ := serverDataJSON["props"].(map[string]any)

	return code, &web.Props{
		Title:       title,
		Description: description,
		Page:        page,
		Data:        props,
	}
}

func toJSONQuery(response *httptest.ResponseRecorder) *jsonq.Query {
	return jsonq.New(response.Body.String())
}
