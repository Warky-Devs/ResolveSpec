package security

import (
	"fmt"
	"net/http"

	"github.com/bitechdev/ResolveSpec/pkg/restheadspec"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// SetupSecurityProvider initializes and configures the security provider
// This should be called when setting up your HTTP server
//
// IMPORTANT: You MUST configure the callbacks before calling this function:
//   - GlobalSecurity.AuthenticateCallback
//   - GlobalSecurity.LoadColumnSecurityCallback
//   - GlobalSecurity.LoadRowSecurityCallback
//
// Example usage in your main.go or server setup:
//
//	// Step 1: Configure callbacks (REQUIRED)
//	security.GlobalSecurity.AuthenticateCallback = myAuthFunction
//	security.GlobalSecurity.LoadColumnSecurityCallback = myLoadColumnSecurityFunction
//	security.GlobalSecurity.LoadRowSecurityCallback = myLoadRowSecurityFunction
//
//	// Step 2: Setup security provider
//	handler := restheadspec.NewHandlerWithGORM(db)
//	security.SetupSecurityProvider(handler, &security.GlobalSecurity)
//
//	// Step 3: Apply middleware
//	router.Use(mux.MiddlewareFunc(security.AuthMiddleware))
//	router.Use(mux.MiddlewareFunc(security.SetSecurityMiddleware))
//
func SetupSecurityProvider(handler *restheadspec.Handler, securityList *SecurityList) error {
	// Validate that required callbacks are configured
	if securityList.AuthenticateCallback == nil {
		return fmt.Errorf("AuthenticateCallback must be set before calling SetupSecurityProvider")
	}
	if securityList.LoadColumnSecurityCallback == nil {
		return fmt.Errorf("LoadColumnSecurityCallback must be set before calling SetupSecurityProvider")
	}
	if securityList.LoadRowSecurityCallback == nil {
		return fmt.Errorf("LoadRowSecurityCallback must be set before calling SetupSecurityProvider")
	}

	// Initialize security maps if needed
	if securityList.ColumnSecurity == nil {
		securityList.ColumnSecurity = make(map[string][]ColumnSecurity)
	}
	if securityList.RowSecurity == nil {
		securityList.RowSecurity = make(map[string]RowSecurity)
	}

	// Register all security hooks
	RegisterSecurityHooks(handler, securityList)

	return nil
}

// Chain creates a middleware chain
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// CompleteExample shows a full integration example with Gorilla Mux
func CompleteExample(db *gorm.DB) (http.Handler, error) {
	// Step 1: Create the ResolveSpec handler
	handler := restheadspec.NewHandlerWithGORM(db)

	// Step 2: Register your models
	// handler.RegisterModel("public", "users", User{})
	// handler.RegisterModel("public", "orders", Order{})

	// Step 3: Configure security callbacks (REQUIRED!)
	// See callbacks_example.go for example implementations
	GlobalSecurity.AuthenticateCallback = ExampleAuthenticateFromHeader
	GlobalSecurity.LoadColumnSecurityCallback = ExampleLoadColumnSecurityFromDatabase
	GlobalSecurity.LoadRowSecurityCallback = ExampleLoadRowSecurityFromDatabase

	// Step 4: Setup security provider
	if err := SetupSecurityProvider(handler, &GlobalSecurity); err != nil {
		return nil, fmt.Errorf("failed to setup security: %v", err)
	}

	// Step 5: Create Mux router and setup routes
	router := mux.NewRouter()

	// The routes are set up by restheadspec, which handles the conversion
	// from http.Request to the internal request format
	restheadspec.SetupMuxRoutes(router, handler)

	// Step 6: Apply middleware to the entire router
	secureRouter := Chain(
		AuthMiddleware,        // Extract user from token
		SetSecurityMiddleware, // Add security context
	)(router)

	return secureRouter, nil
}

// ExampleWithMux shows a simpler integration with Mux
func ExampleWithMux(db *gorm.DB) (*mux.Router, error) {
	handler := restheadspec.NewHandlerWithGORM(db)

	// IMPORTANT: Configure callbacks BEFORE SetupSecurityProvider
	GlobalSecurity.AuthenticateCallback = ExampleAuthenticateFromHeader
	GlobalSecurity.LoadColumnSecurityCallback = ExampleLoadColumnSecurityFromConfig
	GlobalSecurity.LoadRowSecurityCallback = ExampleLoadRowSecurityFromConfig

	if err := SetupSecurityProvider(handler, &GlobalSecurity); err != nil {
		return nil, fmt.Errorf("failed to setup security: %v", err)
	}

	router := mux.NewRouter()

	// Setup API routes
	restheadspec.SetupMuxRoutes(router, handler)

	// Apply middleware to router
	router.Use(mux.MiddlewareFunc(AuthMiddleware))
	router.Use(mux.MiddlewareFunc(SetSecurityMiddleware))

	return router, nil
}

// Example with Gin
// import "github.com/gin-gonic/gin"
//
// func ExampleWithGin(db *gorm.DB) *gin.Engine {
//     handler := restheadspec.NewHandlerWithGORM(db)
//     SetupSecurityProvider(handler, &GlobalSecurity)
//
//     router := gin.Default()
//
//     // Convert middleware to Gin middleware
//     router.Use(func(c *gin.Context) {
//         AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//             c.Request = r
//             c.Next()
//         })).ServeHTTP(c.Writer, c.Request)
//     })
//
//     // Setup routes
//     api := router.Group("/api")
//     api.Any("/:schema/:entity", gin.WrapH(http.HandlerFunc(handler.Handle)))
//     api.Any("/:schema/:entity/:id", gin.WrapH(http.HandlerFunc(handler.Handle)))
//
//     return router
// }
