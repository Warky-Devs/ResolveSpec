package resolvespec

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
	"gorm.io/gorm"
)

// Read handler
func (h *APIHandler) handleRead(w http.ResponseWriter, r *http.Request, schema, entity, id string, options RequestOptions) {
	logger.Info("Reading records from %s.%s", schema, entity)

	// Get the model struct for the entity
	model, err := h.getModelForEntity(schema, entity)
	if err != nil {
		logger.Error("Invalid entity: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_entity", "Invalid entity", err)
		return
	}

	GormTableNameInterface, ok := model.(GormTableNameInterface)
	if !ok {
		logger.Error("Model does not implement GormTableNameInterface")
		h.sendError(w, http.StatusInternalServerError, "model_error", "Model does not implement GormTableNameInterface", nil)
		return
	}
	query := h.db.Model(model).Table(GormTableNameInterface.TableName())

	// Apply column selection
	if len(options.Columns) > 0 {
		logger.Debug("Selecting columns: %v", options.Columns)
		query = query.Select(options.Columns)
	}

	// Apply preloading
	for _, preload := range options.Preload {
		logger.Debug("Applying preload for relation: %s", preload.Relation)
		query = query.Preload(preload.Relation, func(db *gorm.DB) *gorm.DB {

			if len(preload.Columns) > 0 {
				db = db.Select(preload.Columns)
			}
			if len(preload.Filters) > 0 {
				for _, filter := range preload.Filters {
					db = h.applyFilter(db, filter)
				}
			}
			return db
		})

	}

	// Apply filters
	for _, filter := range options.Filters {
		logger.Debug("Applying filter: %s %s %v", filter.Column, filter.Operator, filter.Value)
		query = h.applyFilter(query, filter)
	}

	// Apply sorting
	for _, sort := range options.Sort {
		direction := "ASC"
		if strings.ToLower(sort.Direction) == "desc" {
			direction = "DESC"
		}
		logger.Debug("Applying sort: %s %s", sort.Column, direction)
		query = query.Order(fmt.Sprintf("%s %s", sort.Column, direction))
	}

	// Get total count before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Error("Error counting records: %v", err)
		h.sendError(w, http.StatusInternalServerError, "query_error", "Error counting records", err)
		return
	}
	logger.Debug("Total records before filtering: %d", total)

	// Apply pagination
	if options.Limit != nil && *options.Limit > 0 {
		logger.Debug("Applying limit: %d", *options.Limit)
		query = query.Limit(*options.Limit)
	}
	if options.Offset != nil && *options.Offset > 0 {
		logger.Debug("Applying offset: %d", *options.Offset)
		query = query.Offset(*options.Offset)
	}

	// Execute query
	var result interface{}
	if id != "" {
		logger.Debug("Querying single record with ID: %s", id)
		singleResult := model
		if err := query.First(singleResult, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				logger.Warn("Record not found with ID: %s", id)
				h.sendError(w, http.StatusNotFound, "not_found", "Record not found", nil)
				return
			}
			logger.Error("Error querying record: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
			return
		}
		result = singleResult
	} else {
		logger.Debug("Querying multiple records")
		sliceType := reflect.SliceOf(reflect.TypeOf(model))
		results := reflect.New(sliceType).Interface()

		if err := query.Find(results).Error; err != nil {
			logger.Error("Error querying records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
			return
		}
		result = reflect.ValueOf(results).Elem().Interface()
	}

	logger.Info("Successfully retrieved records")
	h.sendResponse(w, result, &Metadata{
		Total:    total,
		Filtered: total,
		Limit:    optionalInt(options.Limit),
		Offset:   optionalInt(options.Offset),
	})
}

// Create handler
func (h *APIHandler) handleCreate(w http.ResponseWriter, r *http.Request, schema, entity string, data any, options RequestOptions) {
	logger.Info("Creating records for %s.%s", schema, entity)
	query := h.db.Table(fmt.Sprintf("%s.%s", schema, entity))

	switch v := data.(type) {
	case map[string]interface{}:
		result := query.Create(v)
		if result.Error != nil {
			logger.Error("Error creating record: %v", result.Error)
			h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating record", result.Error)
			return
		}
		logger.Info("Successfully created record")
		h.sendResponse(w, v, nil)

	case []map[string]interface{}:
		result := query.Create(v)
		if result.Error != nil {
			logger.Error("Error creating records: %v", result.Error)
			h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating records", result.Error)
			return
		}
		logger.Info("Successfully created %d records", len(v))
		h.sendResponse(w, v, nil)
	case []interface{}:
		list := make([]interface{}, 0)
		for _, item := range v {
			result := query.Create(item)
			list = append(list, item)
			if result.Error != nil {
				logger.Error("Error creating records: %v", result.Error)
				h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating records", result.Error)
				return
			}
			logger.Info("Successfully created %d records", len(v))
		}
		h.sendResponse(w, list, nil)
	default:
		logger.Error("Invalid data type for create operation: %T", data)
	}
}

// Update handler
func (h *APIHandler) handleUpdate(w http.ResponseWriter, r *http.Request, schema, entity string, urlID string, reqID any, data any, options RequestOptions) {
	logger.Info("Updating records for %s.%s", schema, entity)
	query := h.db.Table(fmt.Sprintf("%s.%s", schema, entity))

	switch {
	case urlID != "":
		logger.Debug("Updating by URL ID: %s", urlID)
		result := query.Where("id = ?", urlID).Updates(data)
		handleUpdateResult(w, h, result, data)

	case reqID != nil:
		switch id := reqID.(type) {
		case string:
			logger.Debug("Updating by request ID: %s", id)
			result := query.Where("id = ?", id).Updates(data)
			handleUpdateResult(w, h, result, data)

		case []string:
			logger.Debug("Updating by multiple IDs: %v", id)
			result := query.Where("id IN ?", id).Updates(data)
			handleUpdateResult(w, h, result, data)
		}

	case data != nil:
		switch v := data.(type) {
		case []map[string]interface{}:
			logger.Debug("Performing bulk update with %d records", len(v))
			err := h.db.Transaction(func(tx *gorm.DB) error {
				for _, item := range v {
					if id, ok := item["id"].(string); ok {
						if err := tx.Where("id = ?", id).Updates(item).Error; err != nil {
							logger.Error("Error in bulk update transaction: %v", err)
							return err
						}
					}
				}
				return nil
			})
			if err != nil {
				h.sendError(w, http.StatusInternalServerError, "update_error", "Error in bulk update", err)
				return
			}
			logger.Info("Bulk update completed successfully")
			h.sendResponse(w, data, nil)
		}
	default:
		logger.Error("Invalid data type for update operation: %T", data)

	}
}

// Delete handler
func (h *APIHandler) handleDelete(w http.ResponseWriter, r *http.Request, schema, entity, id string) {
	logger.Info("Deleting records from %s.%s", schema, entity)
	query := h.db.Table(fmt.Sprintf("%s.%s", schema, entity))

	if id == "" {
		logger.Error("Delete operation requires an ID")
		h.sendError(w, http.StatusBadRequest, "missing_id", "Delete operation requires an ID", nil)
		return
	}

	result := query.Delete("id = ?", id)
	if result.Error != nil {
		logger.Error("Error deleting record: %v", result.Error)
		h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting record", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		logger.Warn("No record found to delete with ID: %s", id)
		h.sendError(w, http.StatusNotFound, "not_found", "Record not found", nil)
		return
	}

	logger.Info("Successfully deleted record with ID: %s", id)
	h.sendResponse(w, nil, nil)
}
