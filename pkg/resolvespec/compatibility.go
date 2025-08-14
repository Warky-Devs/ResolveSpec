package resolvespec

import (
	"net/http"

	"github.com/Warky-Devs/ResolveSpec/pkg/models"
	"gorm.io/gorm"
)

// NewAPIHandler creates a new APIHandler instance (backward compatibility)
// For now, this returns the legacy APIHandler to maintain full compatibility
// including preloading functionality. Users can opt-in to new abstractions when ready.
func NewAPIHandler(db *gorm.DB) *APIHandlerCompat {
	legacyHandler := NewLegacyAPIHandler(db)
	
	// Initialize new abstractions for future use
	gormAdapter := NewGormAdapter(db)
	registry := NewModelRegistry()
	
	// Initialize registry with existing models
	models.IterateModels(func(name string, model interface{}) {
		registry.RegisterModel(name, model)
	})
	
	newHandler := NewHandler(gormAdapter, registry)
	
	return &APIHandlerCompat{
		legacyHandler: legacyHandler,
		newHandler:    newHandler,
		db:           db,
	}
}

// APIHandlerCompat provides backward compatibility with the original APIHandler
type APIHandlerCompat struct {
	legacyHandler *LegacyAPIHandler // For full backward compatibility
	newHandler    *Handler          // New abstracted handler (optional use)
	db           *gorm.DB          // Legacy GORM reference
}

// Handle maintains the original signature for backward compatibility
func (a *APIHandlerCompat) Handle(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// Use legacy handler to maintain full compatibility including preloading
	a.legacyHandler.Handle(w, r, params)
}

// HandleGet maintains the original signature for backward compatibility
func (a *APIHandlerCompat) HandleGet(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// Use legacy handler for metadata
	a.legacyHandler.HandleGet(w, r, params)
}

// RegisterModel maintains the original signature for backward compatibility
func (a *APIHandlerCompat) RegisterModel(schema, name string, model interface{}) error {
	// Register with both legacy handler and new handler
	err1 := a.legacyHandler.RegisterModel(schema, name, model)
	err2 := a.newHandler.RegisterModel(schema, name, model)
	if err1 != nil {
		return err1
	}
	return err2
}

// GetNewHandler returns the new abstracted handler for advanced use cases
func (a *APIHandlerCompat) GetNewHandler() *Handler {
	return a.newHandler
}

// GetLegacyHandler returns the legacy handler for cases needing full GORM features
func (a *APIHandlerCompat) GetLegacyHandler() *LegacyAPIHandler {
	return a.legacyHandler
}