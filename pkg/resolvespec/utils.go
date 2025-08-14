package resolvespec

import (
	"fmt"
	"net/http"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
	"github.com/Warky-Devs/ResolveSpec/pkg/models"
	"gorm.io/gorm"
)

func handleUpdateResult(w http.ResponseWriter, h *LegacyAPIHandler, result *gorm.DB, data interface{}) {
	if result.Error != nil {
		logger.Error("Update error: %v", result.Error)
		h.sendError(w, http.StatusInternalServerError, "update_error", "Error updating record(s)", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		logger.Warn("No records found to update")
		h.sendError(w, http.StatusNotFound, "not_found", "No records found to update", nil)
		return
	}
	logger.Info("Successfully updated %d records", result.RowsAffected)
	h.sendResponse(w, data, nil)
}

func optionalInt(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// Helper methods
func (h *LegacyAPIHandler) applyFilter(query *gorm.DB, filter FilterOption) *gorm.DB {
	switch filter.Operator {
	case "eq":
		return query.Where(fmt.Sprintf("%s = ?", filter.Column), filter.Value)
	case "neq":
		return query.Where(fmt.Sprintf("%s != ?", filter.Column), filter.Value)
	case "gt":
		return query.Where(fmt.Sprintf("%s > ?", filter.Column), filter.Value)
	case "gte":
		return query.Where(fmt.Sprintf("%s >= ?", filter.Column), filter.Value)
	case "lt":
		return query.Where(fmt.Sprintf("%s < ?", filter.Column), filter.Value)
	case "lte":
		return query.Where(fmt.Sprintf("%s <= ?", filter.Column), filter.Value)
	case "like":
		return query.Where(fmt.Sprintf("%s LIKE ?", filter.Column), filter.Value)
	case "ilike":
		return query.Where(fmt.Sprintf("%s ILIKE ?", filter.Column), filter.Value)
	case "in":
		return query.Where(fmt.Sprintf("%s IN (?)", filter.Column), filter.Value)
	default:
		return query
	}
}

func (h *LegacyAPIHandler) getModelForEntity(schema, name string) (interface{}, error) {
	model, err := models.GetModelByName(fmt.Sprintf("%s.%s", schema, name))

	if err != nil {
		model, err = models.GetModelByName(name)
	}
	return model, err
}

func (h *LegacyAPIHandler) RegisterModel(schema, name string, model interface{}) error {
	fullname := fmt.Sprintf("%s.%s", schema, name)
	oldModel, err := models.GetModelByName(fullname)
	if oldModel != nil && err != nil {
		return fmt.Errorf("model %s already exists", fullname)
	}
	err = models.RegisterModel(model, fullname)

	return err
}
