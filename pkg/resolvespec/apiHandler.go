package resolvespec

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
	"gorm.io/gorm"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)

type APIHandler struct {
	db *gorm.DB
}

// NewAPIHandler creates a new API handler instance
func NewAPIHandler(db *gorm.DB) *APIHandler {
	return &APIHandler{
		db: db,
	}
}

// Main handler method
func (h *APIHandler) Handle(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode request body: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", err)
		return
	}

	schema := params["schema"]
	entity := params["entity"]
	id := params["id"]

	logger.Info("Handling %s operation for %s.%s", req.Operation, schema, entity)

	switch req.Operation {
	case "read":
		h.handleRead(w, r, schema, entity, id, req.Options)
	case "create":
		h.handleCreate(w, r, schema, entity, req.Data, req.Options)
	case "update":
		h.handleUpdate(w, r, schema, entity, id, req.ID, req.Data, req.Options)
	case "delete":
		h.handleDelete(w, r, schema, entity, id)
	default:
		logger.Error("Invalid operation: %s", req.Operation)
		h.sendError(w, http.StatusBadRequest, "invalid_operation", "Invalid operation", nil)
	}
}

func (h *APIHandler) sendResponse(w http.ResponseWriter, data interface{}, metadata *Metadata) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	})
}

func (h *APIHandler) sendError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
			Detail:  fmt.Sprintf("%v", details),
		},
	})
}
