package resolvespec

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/uptrace/bun"
	"gorm.io/gorm"
)

// NewAPIHandler creates a new APIHandler with GORM (backward compatibility)
func NewAPIHandlerWithGORM(db *gorm.DB) *APIHandlerCompat {
	return NewAPIHandler(db)
}

// NewHandlerWithGORM creates a new Handler with GORM adapter
func NewHandlerWithGORM(db *gorm.DB) *Handler {
	gormAdapter := NewGormAdapter(db)
	registry := NewModelRegistry()
	return NewHandler(gormAdapter, registry)
}

// NewStandardRouter creates a router with standard HTTP handlers
func NewStandardRouter() *StandardMuxAdapter {
	return NewStandardMuxAdapter()
}

// SetupRoutes sets up routes for the ResolveSpec API with backward compatibility
func SetupRoutes(router *mux.Router, handler *APIHandlerCompat) {
	router.HandleFunc("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler.Handle(w, r, vars)
	}).Methods("POST")

	router.HandleFunc("/{schema}/{entity}/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler.Handle(w, r, vars)
	}).Methods("POST")

	router.HandleFunc("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		handler.HandleGet(w, r, vars)
	}).Methods("GET")
}

// Example usage functions for documentation:

// ExampleWithGORM shows how to use ResolveSpec with GORM (current default)
func ExampleWithGORM(db *gorm.DB) {
	// Create handler using GORM (backward compatible)
	handler := NewAPIHandlerWithGORM(db)
	
	// Setup router
	router := mux.NewRouter()
	SetupRoutes(router, handler)
	
	// Register models
	// handler.RegisterModel("public", "users", &User{})
}

// ExampleWithNewAPI shows how to use the new abstracted API
func ExampleWithNewAPI(db *gorm.DB) {
	// Create database adapter
	dbAdapter := NewGormAdapter(db)
	
	// Create model registry
	registry := NewModelRegistry()
	// registry.RegisterModel("public.users", &User{})
	
	// Create handler with new API
	handler := NewHandler(dbAdapter, registry)
	
	// Create router adapter
	routerAdapter := NewStandardRouter()
	
	// Register routes using new API
	routerAdapter.RegisterRoute("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		reqAdapter := NewHTTPRequest(r)
		respAdapter := NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
	})
}

// ExampleWithBun shows how to switch to Bun ORM
func ExampleWithBun(bunDB *bun.DB) {
	// Create Bun adapter
	dbAdapter := NewBunAdapter(bunDB)
	
	// Create model registry
	registry := NewModelRegistry()
	// registry.RegisterModel("public.users", &User{})
	
	// Create handler
	handler := NewHandler(dbAdapter, registry)
	
	// Setup routes same as with GORM
	router := NewStandardRouter()
	router.RegisterRoute("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		reqAdapter := NewHTTPRequest(r)
		respAdapter := NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
	})
}

// ExampleWithBunRouter shows how to use bunrouter from uptrace
func ExampleWithBunRouter(db *gorm.DB) {
	// Create handler (can use any database adapter)
	handler := NewAPIHandler(db)
	
	// Create bunrouter
	router := NewStandardBunRouterAdapter()
	
	// Setup ResolveSpec routes with bunrouter
	SetupBunRouterWithResolveSpec(router.GetBunRouter(), handler)
	
	// Start server
	// http.ListenAndServe(":8080", router.GetBunRouter())
}

// ExampleBunRouterWithBunDB shows the full uptrace stack (bunrouter + Bun ORM)
func ExampleBunRouterWithBunDB(bunDB *bun.DB) {
	// Create Bun database adapter
	dbAdapter := NewBunAdapter(bunDB)
	
	// Create model registry
	registry := NewModelRegistry()
	// registry.RegisterModel("public.users", &User{})
	
	// Create handler with Bun
	handler := NewHandler(dbAdapter, registry)
	
	// Create compatibility wrapper for existing APIs
	compatHandler := &APIHandlerCompat{
		legacyHandler: nil, // No legacy handler needed
		newHandler:    handler,
		db:           nil, // No GORM dependency
	}
	
	// Create bunrouter
	router := NewStandardBunRouterAdapter()
	
	// Setup ResolveSpec routes
	SetupBunRouterWithResolveSpec(router.GetBunRouter(), compatHandler)
	
	// This gives you the full uptrace stack: bunrouter + Bun ORM
	// http.ListenAndServe(":8080", router.GetBunRouter())
}