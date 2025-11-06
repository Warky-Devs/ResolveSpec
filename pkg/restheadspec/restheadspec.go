package restheadspec

import (
	"net/http"

	"github.com/Warky-Devs/ResolveSpec/pkg/common/adapters/database"
	"github.com/Warky-Devs/ResolveSpec/pkg/common/adapters/router"
	"github.com/Warky-Devs/ResolveSpec/pkg/modelregistry"
	"github.com/gorilla/mux"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gorm.io/gorm"
)

// NewHandlerWithGORM creates a new Handler with GORM adapter
func NewHandlerWithGORM(db *gorm.DB) *Handler {
	gormAdapter := database.NewGormAdapter(db)
	registry := modelregistry.NewModelRegistry()
	return NewHandler(gormAdapter, registry)
}

// NewHandlerWithBun creates a new Handler with Bun adapter
func NewHandlerWithBun(db *bun.DB) *Handler {
	bunAdapter := database.NewBunAdapter(db)
	registry := modelregistry.NewModelRegistry()
	return NewHandler(bunAdapter, registry)
}

// NewStandardMuxRouter creates a router with standard Mux HTTP handlers
func NewStandardMuxRouter() *router.StandardMuxAdapter {
	return router.NewStandardMuxAdapter()
}

// NewStandardBunRouter creates a router with standard BunRouter handlers
func NewStandardBunRouter() *router.StandardBunRouterAdapter {
	return router.NewStandardBunRouterAdapter()
}

// SetupMuxRoutes sets up routes for the RestHeadSpec API with Mux
func SetupMuxRoutes(muxRouter *mux.Router, handler *Handler) {
	// GET, POST, PUT, PATCH, DELETE for /{schema}/{entity}
	muxRouter.HandleFunc("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		reqAdapter := router.NewHTTPRequest(r)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, vars)
	}).Methods("GET", "POST")

	// GET, PUT, PATCH, DELETE for /{schema}/{entity}/{id}
	muxRouter.HandleFunc("/{schema}/{entity}/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		reqAdapter := router.NewHTTPRequest(r)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, vars)
	}).Methods("GET", "PUT", "PATCH", "DELETE")

	// GET for metadata (using HandleGet)
	muxRouter.HandleFunc("/{schema}/{entity}/metadata", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		reqAdapter := router.NewHTTPRequest(r)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.HandleGet(respAdapter, reqAdapter, vars)
	}).Methods("GET")
}

// Example usage functions for documentation:

// ExampleWithGORM shows how to use RestHeadSpec with GORM
func ExampleWithGORM(db *gorm.DB) {
	// Create handler using GORM
	handler := NewHandlerWithGORM(db)

	// Setup router
	muxRouter := mux.NewRouter()
	SetupMuxRoutes(muxRouter, handler)

	// Register models
	// handler.registry.RegisterModel("public.users", &User{})
}

// ExampleWithBun shows how to switch to Bun ORM
func ExampleWithBun(bunDB *bun.DB) {
	// Create Bun adapter
	dbAdapter := database.NewBunAdapter(bunDB)

	// Create model registry
	registry := modelregistry.NewModelRegistry()
	// registry.RegisterModel("public.users", &User{})

	// Create handler
	handler := NewHandler(dbAdapter, registry)

	// Setup routes
	muxRouter := mux.NewRouter()
	SetupMuxRoutes(muxRouter, handler)
}

// SetupBunRouterRoutes sets up bunrouter routes for the RestHeadSpec API
func SetupBunRouterRoutes(bunRouter *router.StandardBunRouterAdapter, handler *Handler) {
	r := bunRouter.GetBunRouter()

	// GET and POST for /:schema/:entity
	r.Handle("GET", "/:schema/:entity", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
		return nil
	})

	r.Handle("POST", "/:schema/:entity", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
		return nil
	})

	// GET, PUT, PATCH, DELETE for /:schema/:entity/:id
	r.Handle("GET", "/:schema/:entity/:id", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
			"id":     req.Param("id"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
		return nil
	})

	r.Handle("PUT", "/:schema/:entity/:id", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
			"id":     req.Param("id"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
		return nil
	})

	r.Handle("PATCH", "/:schema/:entity/:id", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
			"id":     req.Param("id"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
		return nil
	})

	r.Handle("DELETE", "/:schema/:entity/:id", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
			"id":     req.Param("id"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.Handle(respAdapter, reqAdapter, params)
		return nil
	})

	// Metadata endpoint
	r.Handle("GET", "/:schema/:entity/metadata", func(w http.ResponseWriter, req bunrouter.Request) error {
		params := map[string]string{
			"schema": req.Param("schema"),
			"entity": req.Param("entity"),
		}
		reqAdapter := router.NewBunRouterRequest(req)
		respAdapter := router.NewHTTPResponseWriter(w)
		handler.HandleGet(respAdapter, reqAdapter, params)
		return nil
	})
}

// ExampleBunRouterWithBunDB shows usage with both BunRouter and Bun DB
func ExampleBunRouterWithBunDB(bunDB *bun.DB) {
	// Create handler
	handler := NewHandlerWithBun(bunDB)

	// Create BunRouter adapter
	routerAdapter := NewStandardBunRouter()

	// Setup routes
	SetupBunRouterRoutes(routerAdapter, handler)

	// Get the underlying router for server setup
	r := routerAdapter.GetBunRouter()

	// Start server
	http.ListenAndServe(":8080", r)
}
