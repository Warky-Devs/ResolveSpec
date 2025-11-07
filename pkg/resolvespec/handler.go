package resolvespec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/Warky-Devs/ResolveSpec/pkg/common"
	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
)

// Handler handles API requests using database and model abstractions
type Handler struct {
	db       common.Database
	registry common.ModelRegistry
}

// NewHandler creates a new API handler with database and registry abstractions
func NewHandler(db common.Database, registry common.ModelRegistry) *Handler {
	return &Handler{
		db:       db,
		registry: registry,
	}
}

// handlePanic is a helper function to handle panics with stack traces
func (h *Handler) handlePanic(w common.ResponseWriter, method string, err interface{}) {
	stack := debug.Stack()
	logger.Error("Panic in %s: %v\nStack trace:\n%s", method, err, string(stack))
	h.sendError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Internal server error in %s", method), fmt.Errorf("%v", err))
}

// Handle processes API requests through router-agnostic interface
func (h *Handler) Handle(w common.ResponseWriter, r common.Request, params map[string]string) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "Handle", err)
		}
	}()

	ctx := context.Background()

	body, err := r.Body()
	if err != nil {
		logger.Error("Failed to read request body: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body", err)
		return
	}

	var req common.RequestBody
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Error("Failed to decode request body: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", err)
		return
	}

	schema := params["schema"]
	entity := params["entity"]
	id := params["id"]

	logger.Info("Handling %s operation for %s.%s", req.Operation, schema, entity)

	// Get model and populate context with request-scoped data
	model, err := h.registry.GetModelByEntity(schema, entity)
	if err != nil {
		logger.Error("Invalid entity: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_entity", "Invalid entity", err)
		return
	}

	// Validate that the model is a struct type (not a slice or pointer to slice)
	modelType := reflect.TypeOf(model)
	originalType := modelType
	for modelType != nil && (modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) {
		modelType = modelType.Elem()
	}

	if modelType == nil || modelType.Kind() != reflect.Struct {
		logger.Error("Model for %s.%s must be a struct type, got %v. Please register models as struct types, not slices or pointers to slices.", schema, entity, originalType)
		h.sendError(w, http.StatusInternalServerError, "invalid_model_type",
			fmt.Sprintf("Model must be a struct type, got %v. Ensure you register the struct (e.g., ModelCoreAccount{}) not a slice (e.g., []*ModelCoreAccount)", originalType),
			fmt.Errorf("invalid model type: %v", originalType))
		return
	}

	// If the registered model was a pointer or slice, use the unwrapped struct type
	if originalType != modelType {
		model = reflect.New(modelType).Elem().Interface()
	}

	// Create a pointer to the model type for database operations
	modelPtr := reflect.New(reflect.TypeOf(model)).Interface()
	tableName := h.getTableName(schema, entity, model)

	// Add request-scoped data to context
	ctx = WithRequestData(ctx, schema, entity, tableName, model, modelPtr)

	switch req.Operation {
	case "read":
		h.handleRead(ctx, w, id, req.Options)
	case "create":
		h.handleCreate(ctx, w, req.Data, req.Options)
	case "update":
		h.handleUpdate(ctx, w, id, req.ID, req.Data, req.Options)
	case "delete":
		h.handleDelete(ctx, w, id)
	default:
		logger.Error("Invalid operation: %s", req.Operation)
		h.sendError(w, http.StatusBadRequest, "invalid_operation", "Invalid operation", nil)
	}
}

// HandleGet processes GET requests for metadata
func (h *Handler) HandleGet(w common.ResponseWriter, r common.Request, params map[string]string) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "HandleGet", err)
		}
	}()

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

func (h *Handler) handleRead(ctx context.Context, w common.ResponseWriter, id string, options common.RequestOptions) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleRead", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)
	model := GetModel(ctx)

	// Validate and unwrap model type to get base struct
	modelType := reflect.TypeOf(model)
	for modelType != nil && (modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) {
		modelType = modelType.Elem()
	}

	if modelType == nil || modelType.Kind() != reflect.Struct {
		logger.Error("Model must be a struct type, got %v for %s.%s", modelType, schema, entity)
		h.sendError(w, http.StatusInternalServerError, "invalid_model", "Model must be a struct type", fmt.Errorf("invalid model type: %v", modelType))
		return
	}

	// Create a pointer to the model type for database operations
	modelPtr := reflect.New(modelType).Interface()

	logger.Info("Reading records from %s.%s", schema, entity)

	query := h.db.NewSelect().Model(modelPtr)
	query = query.Table(tableName)

	// Apply column selection
	if len(options.Columns) > 0 {
		logger.Debug("Selecting columns: %v", options.Columns)
		query = query.Column(options.Columns...)
	}

	// Apply preloading
	if len(options.Preload) > 0 {
		query = h.applyPreloads(model, query, options.Preload)
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
		// Create a pointer to the struct type for scanning - use modelType which is already unwrapped
		singleResult := reflect.New(modelType).Interface()
		query = query.Where("id = ?", id)
		if err := query.Scan(ctx, singleResult); err != nil {
			logger.Error("Error querying record: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
			return
		}
		result = singleResult
	} else {
		logger.Debug("Querying multiple records")
		// Create a slice of pointers to the model type - use modelType which is already unwrapped
		sliceType := reflect.SliceOf(reflect.PointerTo(modelType))
		results := reflect.New(sliceType).Interface()

		if err := query.Scan(ctx, results); err != nil {
			logger.Error("Error querying records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
			return
		}
		result = reflect.ValueOf(results).Elem().Interface()
	}

	logger.Info("Successfully retrieved records")

	limit := 0
	if options.Limit != nil {
		limit = *options.Limit
	}
	offset := 0
	if options.Offset != nil {
		offset = *options.Offset
	}

	h.sendResponse(w, result, &common.Metadata{
		Total:    int64(total),
		Filtered: int64(total),
		Limit:    limit,
		Offset:   offset,
	})
}

func (h *Handler) handleCreate(ctx context.Context, w common.ResponseWriter, data interface{}, options common.RequestOptions) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleCreate", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)

	logger.Info("Creating records for %s.%s", schema, entity)

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
		err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
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
		err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
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

func (h *Handler) handleUpdate(ctx context.Context, w common.ResponseWriter, urlID string, reqID interface{}, data interface{}, options common.RequestOptions) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleUpdate", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)

	logger.Info("Updating records for %s.%s", schema, entity)

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

func (h *Handler) handleDelete(ctx context.Context, w common.ResponseWriter, id string) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleDelete", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)

	logger.Info("Deleting records from %s.%s", schema, entity)

	if id == "" {
		logger.Error("Delete operation requires an ID")
		h.sendError(w, http.StatusBadRequest, "missing_id", "Delete operation requires an ID", nil)
		return
	}

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

func (h *Handler) applyFilter(query common.SelectQuery, filter common.FilterOption) common.SelectQuery {
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
	if provider, ok := model.(common.TableNameProvider); ok {
		return provider.TableName()
	}
	return fmt.Sprintf("%s.%s", schema, entity)
}

func (h *Handler) generateMetadata(schema, entity string, model interface{}) *common.TableMetadata {
	modelType := reflect.TypeOf(model)

	// Unwrap pointers, slices, and arrays to get to the base struct type
	for modelType != nil && (modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) {
		modelType = modelType.Elem()
	}

	// Validate that we have a struct type
	if modelType == nil || modelType.Kind() != reflect.Struct {
		logger.Error("Model type must be a struct, got %v for %s.%s", modelType, schema, entity)
		return &common.TableMetadata{
			Schema:    schema,
			Table:     entity,
			Columns:   make([]common.Column, 0),
			Relations: make([]string, 0),
		}
	}

	metadata := &common.TableMetadata{
		Schema:    schema,
		Table:     entity,
		Columns:   make([]common.Column, 0),
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

		column := common.Column{
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

func (h *Handler) sendResponse(w common.ResponseWriter, data interface{}, metadata *common.Metadata) {
	w.SetHeader("Content-Type", "application/json")
	w.WriteJSON(common.Response{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	})
}

func (h *Handler) sendError(w common.ResponseWriter, status int, code, message string, details interface{}) {
	w.SetHeader("Content-Type", "application/json")
	w.WriteHeader(status)
	w.WriteJSON(common.Response{
		Success: false,
		Error: &common.APIError{
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

// Helper functions

func getColumnType(field reflect.StructField) string {
	// Check GORM type tag first
	gormTag := field.Tag.Get("gorm")
	if strings.Contains(gormTag, "type:") {
		parts := strings.Split(gormTag, "type:")
		if len(parts) > 1 {
			typePart := strings.Split(parts[1], ";")[0]
			return typePart
		}
	}

	// Map Go types to SQL types
	switch field.Type.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int32:
		return "integer"
	case reflect.Int64:
		return "bigint"
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	case reflect.Bool:
		return "boolean"
	default:
		if field.Type.Name() == "Time" {
			return "timestamp"
		}
		return "unknown"
	}
}

func isNullable(field reflect.StructField) bool {
	// Check if it's a pointer type
	if field.Type.Kind() == reflect.Ptr {
		return true
	}

	// Check if it's a null type from sql package
	typeName := field.Type.Name()
	if strings.HasPrefix(typeName, "Null") {
		return true
	}

	// Check GORM tags
	gormTag := field.Tag.Get("gorm")
	return !strings.Contains(gormTag, "not null")
}

// Preload support functions

type relationshipInfo struct {
	fieldName    string
	jsonName     string
	relationType string // "belongsTo", "hasMany", "hasOne", "many2many"
	foreignKey   string
	references   string
	joinTable    string
	relatedModel interface{}
}

func (h *Handler) applyPreloads(model interface{}, query common.SelectQuery, preloads []common.PreloadOption) common.SelectQuery {
	modelType := reflect.TypeOf(model)

	// Unwrap pointers, slices, and arrays to get to the base struct type
	for modelType != nil && (modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) {
		modelType = modelType.Elem()
	}

	// Validate that we have a struct type
	if modelType == nil || modelType.Kind() != reflect.Struct {
		logger.Warn("Cannot apply preloads to non-struct type: %v", modelType)
		return query
	}

	for _, preload := range preloads {
		logger.Debug("Processing preload for relation: %s", preload.Relation)
		relInfo := h.getRelationshipInfo(modelType, preload.Relation)
		if relInfo == nil {
			logger.Warn("Relation %s not found in model", preload.Relation)
			continue
		}

		// Use the field name (capitalized) for ORM preloading
		// ORMs like GORM and Bun expect the struct field name, not the JSON name
		relationFieldName := relInfo.fieldName

		// For now, we'll preload without conditions
		// TODO: Implement column selection and filtering for preloads
		// This requires a more sophisticated approach with callbacks or query builders
		query = query.Preload(relationFieldName)
		logger.Debug("Applied Preload for relation: %s (field: %s)", preload.Relation, relationFieldName)
	}

	return query
}

func (h *Handler) getRelationshipInfo(modelType reflect.Type, relationName string) *relationshipInfo {
	// Ensure we have a struct type
	if modelType == nil || modelType.Kind() != reflect.Struct {
		logger.Warn("Cannot get relationship info from non-struct type: %v", modelType)
		return nil
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		jsonTag := field.Tag.Get("json")
		jsonName := strings.Split(jsonTag, ",")[0]

		if jsonName == relationName {
			gormTag := field.Tag.Get("gorm")
			info := &relationshipInfo{
				fieldName: field.Name,
				jsonName:  jsonName,
			}

			// Parse GORM tag to determine relationship type and keys
			if strings.Contains(gormTag, "foreignKey") {
				info.foreignKey = h.extractTagValue(gormTag, "foreignKey")
				info.references = h.extractTagValue(gormTag, "references")

				// Determine if it's belongsTo or hasMany/hasOne
				if field.Type.Kind() == reflect.Slice {
					info.relationType = "hasMany"
				} else if field.Type.Kind() == reflect.Ptr || field.Type.Kind() == reflect.Struct {
					info.relationType = "belongsTo"
				}
			} else if strings.Contains(gormTag, "many2many") {
				info.relationType = "many2many"
				info.joinTable = h.extractTagValue(gormTag, "many2many")
			}

			return info
		}
	}
	return nil
}

func (h *Handler) extractTagValue(tag, key string) string {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, key+":") {
			return strings.TrimPrefix(part, key+":")
		}
	}
	return ""
}
