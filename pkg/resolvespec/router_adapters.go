package resolvespec

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// MuxAdapter adapts Gorilla Mux to work with our Router interface
type MuxAdapter struct {
	router *mux.Router
}

// NewMuxAdapter creates a new Mux adapter
func NewMuxAdapter(router *mux.Router) *MuxAdapter {
	return &MuxAdapter{router: router}
}

func (m *MuxAdapter) HandleFunc(pattern string, handler HTTPHandlerFunc) RouteRegistration {
	route := &MuxRouteRegistration{
		router:  m.router,
		pattern: pattern,
		handler: handler,
	}
	return route
}

func (m *MuxAdapter) ServeHTTP(w ResponseWriter, r Request) {
	// This method would be used when we need to serve through our interface
	// For now, we'll work directly with the underlying router
	panic("ServeHTTP not implemented - use GetMuxRouter() for direct access")
}

// MuxRouteRegistration implements RouteRegistration for Mux
type MuxRouteRegistration struct {
	router  *mux.Router
	pattern string
	handler HTTPHandlerFunc
	route   *mux.Route
}

func (m *MuxRouteRegistration) Methods(methods ...string) RouteRegistration {
	if m.route == nil {
		m.route = m.router.HandleFunc(m.pattern, func(w http.ResponseWriter, r *http.Request) {
			reqAdapter := &HTTPRequest{req: r, vars: mux.Vars(r)}
			respAdapter := &HTTPResponseWriter{resp: w}
			m.handler(respAdapter, reqAdapter)
		})
	}
	m.route.Methods(methods...)
	return m
}

func (m *MuxRouteRegistration) PathPrefix(prefix string) RouteRegistration {
	if m.route == nil {
		m.route = m.router.HandleFunc(m.pattern, func(w http.ResponseWriter, r *http.Request) {
			reqAdapter := &HTTPRequest{req: r, vars: mux.Vars(r)}
			respAdapter := &HTTPResponseWriter{resp: w}
			m.handler(respAdapter, reqAdapter)
		})
	}
	m.route.PathPrefix(prefix)
	return m
}

// HTTPRequest adapts standard http.Request to our Request interface
type HTTPRequest struct {
	req  *http.Request
	vars map[string]string
	body []byte
}

func NewHTTPRequest(r *http.Request) *HTTPRequest {
	return &HTTPRequest{
		req:  r,
		vars: make(map[string]string),
	}
}

func (h *HTTPRequest) Method() string {
	return h.req.Method
}

func (h *HTTPRequest) URL() string {
	return h.req.URL.String()
}

func (h *HTTPRequest) Header(key string) string {
	return h.req.Header.Get(key)
}

func (h *HTTPRequest) Body() ([]byte, error) {
	if h.body != nil {
		return h.body, nil
	}
	if h.req.Body == nil {
		return nil, nil
	}
	defer h.req.Body.Close()
	body, err := io.ReadAll(h.req.Body)
	if err != nil {
		return nil, err
	}
	h.body = body
	return body, nil
}

func (h *HTTPRequest) PathParam(key string) string {
	return h.vars[key]
}

func (h *HTTPRequest) QueryParam(key string) string {
	return h.req.URL.Query().Get(key)
}

// HTTPResponseWriter adapts our ResponseWriter interface to standard http.ResponseWriter
type HTTPResponseWriter struct {
	resp   http.ResponseWriter
	w      ResponseWriter
	status int
}

func NewHTTPResponseWriter(w http.ResponseWriter) *HTTPResponseWriter {
	return &HTTPResponseWriter{resp: w}
}


func (h *HTTPResponseWriter) SetHeader(key, value string) {
	h.resp.Header().Set(key, value)
}

func (h *HTTPResponseWriter) WriteHeader(statusCode int) {
	h.status = statusCode
	h.resp.WriteHeader(statusCode)
}

func (h *HTTPResponseWriter) Write(data []byte) (int, error) {
	return h.resp.Write(data)
}

func (h *HTTPResponseWriter) WriteJSON(data interface{}) error {
	h.SetHeader("Content-Type", "application/json")
	return json.NewEncoder(h.resp).Encode(data)
}

// StandardMuxAdapter creates routes compatible with standard http.HandlerFunc
type StandardMuxAdapter struct {
	*MuxAdapter
}

func NewStandardMuxAdapter() *StandardMuxAdapter {
	return &StandardMuxAdapter{
		MuxAdapter: NewMuxAdapter(mux.NewRouter()),
	}
}

// RegisterRoute registers a route that works with the existing APIHandler
func (s *StandardMuxAdapter) RegisterRoute(pattern string, handler func(http.ResponseWriter, *http.Request, map[string]string)) *mux.Route {
	return s.router.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler(w, r, vars)
	})
}

// GetMuxRouter returns the underlying mux router for direct access
func (s *StandardMuxAdapter) GetMuxRouter() *mux.Router {
	return s.router
}

// GinAdapter for future Gin support
type GinAdapter struct {
	// This would be implemented when Gin support is needed
	// engine *gin.Engine  
}

// EchoAdapter for future Echo support  
type EchoAdapter struct {
	// This would be implemented when Echo support is needed
	// echo *echo.Echo
}

// PathParamExtractor extracts path parameters from different router types
type PathParamExtractor interface {
	ExtractParams(*http.Request) map[string]string
}

// MuxParamExtractor extracts parameters from Gorilla Mux
type MuxParamExtractor struct{}

func (m MuxParamExtractor) ExtractParams(r *http.Request) map[string]string {
	return mux.Vars(r)
}

// RouterConfig holds router configuration
type RouterConfig struct {
	PathPrefix     string
	Middleware     []func(http.Handler) http.Handler
	ParamExtractor PathParamExtractor
}

// DefaultRouterConfig returns default router configuration
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		PathPrefix:     "",
		Middleware:     make([]func(http.Handler) http.Handler, 0),
		ParamExtractor: MuxParamExtractor{},
	}
}