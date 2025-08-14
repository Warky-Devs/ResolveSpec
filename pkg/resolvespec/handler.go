package resolvespec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
)

// Handler handles API requests using database and model abstractions
type Handler struct {
	db       Database
	registry ModelRegistry
}

// NewHandler creates a new API handler with database and registry abstractions
func NewHandler(db Database, registry ModelRegistry) *Handler {
	return &Handler{
		db:       db,
		registry: registry,
	}
}

// Handle processes API requests through router-agnostic interface
func (h *Handler) Handle(w ResponseWriter, r Request, params map[string]string) {
	ctx := context.Background()
	
	body, err := r.Body()
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body", err)
		return
	}

	var req RequestBody
	if err := json.Unmarshal(body, &req); err != nil {
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
		h.handleRead(ctx, w, schema, entity, id, req.Options)
	case "create":
		h.handleCreate(ctx, w, schema, entity, req.Data, req.Options)
	case "update":
		h.handleUpdate(ctx, w, schema, entity, id, req.ID, req.Data, req.Options)
	case "delete":
		h.handleDelete(ctx, w, schema, entity, id)
	default:
		logger.Error("Invalid operation: %s", req.Operation)
		h.sendError(w, http.StatusBadRequest, "invalid_operation", "Invalid operation", nil)
	}
}

// HandleGet processes GET requests for metadata
func (h *Handler) HandleGet(w ResponseWriter, r Request, params map[string]string) {
	schema := params["schema"]
	entity := params["entity"]

	logger.Info("Getting metadata for %s.%s", schema, entity)

	model, err := h.registry.GetModelByEntity(schema, entity)
	if err != nil {
		logger.Error("Failed to get model: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_entity", "Invalid entity", err)
		return
	}

	metadata := h.generateMetadata(schema, entity, model)
	h.sendResponse(w, metadata, nil)
}

func (h *Handler) handleRead(ctx context.Context, w ResponseWriter, schema, entity, id string, options RequestOptions) {
	logger.Info("Reading records from %s.%s", schema, entity)

	model, err := h.registry.GetModelByEntity(schema, entity)
	if err != nil {
		logger.Error("Invalid entity: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_entity", "Invalid entity", err)
		return
	}

	query := h.db.NewSelect().Model(model)

	// Get table name
	tableName := h.getTableName(schema, entity, model)
	query = query.Table(tableName)

	// Apply column selection
	if len(options.Columns) > 0 {
		logger.Debug("Selecting columns: %v", options.Columns)
		query = query.Column(options.Columns...)
	}

	// Note: Preloading is not implemented in the new database abstraction yet
	// This is a limitation of the current interface design
	// For now, preloading should use the legacy APIHandler
	if len(options.Preload) > 0 {
		logger.Warn("Preloading not yet implemented in new handler - use legacy APIHandler for preload functionality")
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
	total, err := query.Count(ctx)
	if err != nil {
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
		query = query.Where("id = ?", id)
		if err := query.Scan(ctx, singleResult); err != nil {
			logger.Error("Error querying record: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
			return
		}
		result = singleResult
	} else {
		logger.Debug("Querying multiple records")
		sliceType := reflect.SliceOf(reflect.TypeOf(model))
		results := reflect.New(sliceType).Interface()

		if err := query.Scan(ctx, results); err != nil {
			logger.Error("Error querying records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
			return
		}
		result = reflect.ValueOf(results).Elem().Interface()
	}

	logger.Info("Successfully retrieved records")
	h.sendResponse(w, result, &Metadata{
		Total:    int64(total),
		Filtered: int64(total),
		Limit:    optionalInt(options.Limit),
		Offset:   optionalInt(options.Offset),
	})
}

func (h *Handler) handleCreate(ctx context.Context, w ResponseWriter, schema, entity string, data interface{}, options RequestOptions) {
	logger.Info("Creating records for %s.%s", schema, entity)

	tableName := fmt.Sprintf("%s.%s", schema, entity)
	query := h.db.NewInsert().Table(tableName)

	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			query = query.Value(key, value)
		}
		result, err := query.Exec(ctx)
		if err != nil {
			logger.Error("Error creating record: %v", err)
			h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating record", err)
			return
		}
		logger.Info("Successfully created record, rows affected: %d", result.RowsAffected())
		h.sendResponse(w, v, nil)

	case []map[string]interface{}:
		err := h.db.RunInTransaction(ctx, func(tx Database) error {
			for _, item := range v {
				txQuery := tx.NewInsert().Table(tableName)
				for key, value := range item {
					txQuery = txQuery.Value(key, value)
				}
				if _, err := txQuery.Exec(ctx); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			logger.Error("Error creating records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating records", err)
			return
		}
		logger.Info("Successfully created %d records", len(v))
		h.sendResponse(w, v, nil)

	case []interface{}:
		// Handle []interface{} type from JSON unmarshaling
		list := make([]interface{}, 0)
		err := h.db.RunInTransaction(ctx, func(tx Database) error {
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					txQuery := tx.NewInsert().Table(tableName)
					for key, value := range itemMap {
						txQuery = txQuery.Value(key, value)
					}
					if _, err := txQuery.Exec(ctx); err != nil {
						return err
					}
					list = append(list, item)
				}
			}
			return nil
		})
		if err != nil {
			logger.Error("Error creating records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating records", err)
			return
		}
		logger.Info("Successfully created %d records", len(v))
		h.sendResponse(w, list, nil)

	default:
		logger.Error("Invalid data type for create operation: %T", data)
		h.sendError(w, http.StatusBadRequest, "invalid_data", "Invalid data type for create operation", nil)
	}
}

func (h *Handler) handleUpdate(ctx context.Context, w ResponseWriter, schema, entity, urlID string, reqID interface{}, data interface{}, options RequestOptions) {
	logger.Info("Updating records for %s.%s", schema, entity)

	tableName := fmt.Sprintf("%s.%s", schema, entity)
	query := h.db.NewUpdate().Table(tableName)

	switch updates := data.(type) {
	case map[string]interface{}:
		query = query.SetMap(updates)
	default:
		logger.Error("Invalid data type for update operation: %T", data)
		h.sendError(w, http.StatusBadRequest, "invalid_data", "Invalid data type for update operation", nil)
		return
	}

	// Apply conditions
	if urlID != "" {
		logger.Debug("Updating by URL ID: %s", urlID)
		query = query.Where("id = ?", urlID)
	} else if reqID != nil {
		switch id := reqID.(type) {
		case string:
			logger.Debug("Updating by request ID: %s", id)
			query = query.Where("id = ?", id)
		case []string:
			logger.Debug("Updating by multiple IDs: %v", id)
			query = query.Where("id IN (?)", id)
		}
	}

	result, err := query.Exec(ctx)
	if err != nil {
		logger.Error("Update error: %v", err)
		h.sendError(w, http.StatusInternalServerError, "update_error", "Error updating record(s)", err)
		return
	}
	
	if result.RowsAffected() == 0 {
		logger.Warn("No records found to update")
		h.sendError(w, http.StatusNotFound, "not_found", "No records found to update", nil)
		return
	}
	
	logger.Info("Successfully updated %d records", result.RowsAffected())
	h.sendResponse(w, data, nil)
}

func (h *Handler) handleDelete(ctx context.Context, w ResponseWriter, schema, entity, id string) {
	logger.Info("Deleting records from %s.%s", schema, entity)

	if id == "" {
		logger.Error("Delete operation requires an ID")
		h.sendError(w, http.StatusBadRequest, "missing_id", "Delete operation requires an ID", nil)
		return
	}

	tableName := fmt.Sprintf("%s.%s", schema, entity)
	query := h.db.NewDelete().Table(tableName).Where("id = ?", id)

	result, err := query.Exec(ctx)
	if err != nil {
		logger.Error("Error deleting record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting record", err)
		return
	}
	
	if result.RowsAffected() == 0 {
		logger.Warn("No record found to delete with ID: %s", id)
		h.sendError(w, http.StatusNotFound, "not_found", "Record not found", nil)
		return
	}

	logger.Info("Successfully deleted record with ID: %s", id)
	h.sendResponse(w, nil, nil)
}

func (h *Handler) applyFilter(query SelectQuery, filter FilterOption) SelectQuery {
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

func (h *Handler) getTableName(schema, entity string, model interface{}) string {
	if provider, ok := model.(TableNameProvider); ok {
		return provider.TableName()
	}
	return fmt.Sprintf("%s.%s", schema, entity)
}

func (h *Handler) generateMetadata(schema, entity string, model interface{}) TableMetadata {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	metadata := TableMetadata{
		Schema:    schema,
		Table:     entity,
		Columns:   make([]Column, 0),
		Relations: make([]string, 0),
	}

	// Generate metadata using reflection (same logic as before)
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		if !field.IsExported() {
			continue
		}

		gormTag := field.Tag.Get("gorm")
		jsonTag := field.Tag.Get("json")

		if jsonTag == "-" {
			continue
		}

		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		if field.Type.Kind() == reflect.Slice ||
			(field.Type.Kind() == reflect.Struct && field.Type.Name() != "Time") {
			metadata.Relations = append(metadata.Relations, jsonName)
			continue
		}

		column := Column{
			Name:       jsonName,
			Type:       getColumnType(field),
			IsNullable: isNullable(field),
			IsPrimary:  strings.Contains(gormTag, "primaryKey"),
			IsUnique:   strings.Contains(gormTag, "unique") || strings.Contains(gormTag, "uniqueIndex"),
			HasIndex:   strings.Contains(gormTag, "index") || strings.Contains(gormTag, "uniqueIndex"),
		}

		metadata.Columns = append(metadata.Columns, column)
	}

	return metadata
}

func (h *Handler) sendResponse(w ResponseWriter, data interface{}, metadata *Metadata) {
	w.SetHeader("Content-Type", "application/json")
	w.WriteJSON(Response{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	})
}

func (h *Handler) sendError(w ResponseWriter, status int, code, message string, details interface{}) {
	w.SetHeader("Content-Type", "application/json")
	w.WriteHeader(status)
	w.WriteJSON(Response{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
			Detail:  fmt.Sprintf("%v", details),
		},
	})
}

// RegisterModel allows registering models at runtime
func (h *Handler) RegisterModel(schema, name string, model interface{}) error {
	fullname := fmt.Sprintf("%s.%s", schema, name)
	return h.registry.RegisterModel(fullname, model)
}