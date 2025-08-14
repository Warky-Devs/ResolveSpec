package resolvespec

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
	"gorm.io/gorm"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)

type LegacyAPIHandler struct {
	db *gorm.DB
}

// NewLegacyAPIHandler creates a new legacy API handler instance
func NewLegacyAPIHandler(db *gorm.DB) *LegacyAPIHandler {
	return &LegacyAPIHandler{
		db: db,
	}
}

// Main handler method
func (h *LegacyAPIHandler) Handle(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var req RequestBody

	if r.Body == nil {
		logger.Error("No body to decode")
		h.sendError(w, http.StatusBadRequest, "invalid_request", "No body to decode", nil)
		return
	} else {
		defer r.Body.Close()
	}
	if bodyContents, err := io.ReadAll(r.Body); err != nil {
		logger.Error("Failed to decode read body: %v", err)
		h.sendError(w, http.StatusBadRequest, "read_request", "Invalid request body", err)
		return
	} else {
		if err := json.Unmarshal(bodyContents, &req); err != nil {
			logger.Error("Failed to decode request body: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", err)
			return
		}
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

func (h *LegacyAPIHandler) sendResponse(w http.ResponseWriter, data interface{}, metadata *Metadata) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	})
}

func (h *LegacyAPIHandler) sendError(w http.ResponseWriter, status int, code, message string, details interface{}) {
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
