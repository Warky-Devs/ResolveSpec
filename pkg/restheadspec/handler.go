package restheadspec

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
// This handler reads filters, columns, and options from HTTP headers
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
// Options are read from HTTP headers instead of request body
func (h *Handler) Handle(w common.ResponseWriter, r common.Request, params map[string]string) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "Handle", err)
		}
	}()

	ctx := context.Background()

	schema := params["schema"]
	entity := params["entity"]
	id := params["id"]

	// Parse options from headers (now returns ExtendedRequestOptions)
	options := h.parseOptionsFromHeaders(r)

	// Determine operation based on HTTP method
	method := r.Method()

	logger.Info("Handling %s request for %s.%s", method, schema, entity)

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

	modelPtr := reflect.New(reflect.TypeOf(model)).Interface()
	tableName := h.getTableName(schema, entity, model)

	// Add request-scoped data to context
	ctx = WithRequestData(ctx, schema, entity, tableName, model, modelPtr)

	// Validate and filter columns in options (log warnings for invalid columns)
	validator := common.NewColumnValidator(model)
	options = filterExtendedOptions(validator, options)

	switch method {
	case "GET":
		if id != "" {
			// GET with ID - read single record
			h.handleRead(ctx, w, id, options)
		} else {
			// GET without ID - read multiple records
			h.handleRead(ctx, w, "", options)
		}
	case "POST":
		// Create operation
		body, err := r.Body()
		if err != nil {
			logger.Error("Failed to read request body: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body", err)
			return
		}
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			logger.Error("Failed to decode request body: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", err)
			return
		}
		h.handleCreate(ctx, w, data, options)
	case "PUT", "PATCH":
		// Update operation
		body, err := r.Body()
		if err != nil {
			logger.Error("Failed to read request body: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body", err)
			return
		}
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			logger.Error("Failed to decode request body: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", err)
			return
		}
		h.handleUpdate(ctx, w, id, nil, data, options)
	case "DELETE":
		h.handleDelete(ctx, w, id)
	default:
		logger.Error("Invalid HTTP method: %s", method)
		h.sendError(w, http.StatusMethodNotAllowed, "invalid_method", "Invalid HTTP method", nil)
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

// parseOptionsFromHeaders is now implemented in headers.go

func (h *Handler) handleRead(ctx context.Context, w common.ResponseWriter, id string, options ExtendedRequestOptions) {
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

	// Create a pointer to a slice of pointers to the model type for query results
	modelPtr := reflect.New(reflect.SliceOf(reflect.PointerTo(modelType))).Interface()

	logger.Info("Reading records from %s.%s", schema, entity)

	// Use Table() with the resolved table name and Model() for Bun compatibility
	// Bun requires Model() to be set for Count() and Scan() operations
	query := h.db.NewSelect().Model(modelPtr).Table(tableName)

	// Apply column selection
	if len(options.Columns) > 0 {
		logger.Debug("Selecting columns: %v", options.Columns)
		query = query.Column(options.Columns...)
	}

	// Apply preloading
	for _, preload := range options.Preload {
		logger.Debug("Applying preload: %s", preload.Relation)
		query = query.Preload(preload.Relation)
	}

	// Apply expand (LEFT JOIN)
	for _, expand := range options.Expand {
		logger.Debug("Applying expand: %s", expand.Relation)
		// Note: Expand would require JOIN implementation
		// For now, we'll use Preload as a fallback
		query = query.Preload(expand.Relation)
	}

	// Apply DISTINCT if requested
	if options.Distinct {
		logger.Debug("Applying DISTINCT")
		// Note: DISTINCT implementation depends on ORM support
		// This may need to be handled differently per database adapter
	}

	// Apply filters
	for _, filter := range options.Filters {
		logger.Debug("Applying filter: %s %s %v", filter.Column, filter.Operator, filter.Value)
		query = h.applyFilter(query, filter)
	}

	// Apply custom SQL WHERE clause (AND condition)
	if options.CustomSQLWhere != "" {
		logger.Debug("Applying custom SQL WHERE: %s", options.CustomSQLWhere)
		query = query.Where(options.CustomSQLWhere)
	}

	// Apply custom SQL WHERE clause (OR condition)
	if options.CustomSQLOr != "" {
		logger.Debug("Applying custom SQL OR: %s", options.CustomSQLOr)
		query = query.WhereOr(options.CustomSQLOr)
	}

	// If ID is provided, filter by ID
	if id != "" {
		logger.Debug("Filtering by ID: %s", id)
		query = query.Where("id = ?", id)
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

	// Get total count before pagination (unless skip count is requested)
	var total int
	if !options.SkipCount {
		count, err := query.Count(ctx)
		if err != nil {
			logger.Error("Error counting records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "query_error", "Error counting records", err)
			return
		}
		total = count
		logger.Debug("Total records: %d", total)
	} else {
		logger.Debug("Skipping count as requested")
		total = -1 // Indicate count was skipped
	}

	// Apply pagination
	if options.Limit != nil && *options.Limit > 0 {
		logger.Debug("Applying limit: %d", *options.Limit)
		query = query.Limit(*options.Limit)
	}
	if options.Offset != nil && *options.Offset > 0 {
		logger.Debug("Applying offset: %d", *options.Offset)
		query = query.Offset(*options.Offset)
	}

	// Execute query - modelPtr was already created earlier
	if err := query.Scan(ctx, modelPtr); err != nil {
		logger.Error("Error executing query: %v", err)
		h.sendError(w, http.StatusInternalServerError, "query_error", "Error executing query", err)
		return
	}

	limit := 0
	if options.Limit != nil {
		limit = *options.Limit
	}
	offset := 0
	if options.Offset != nil {
		offset = *options.Offset
	}

	metadata := &common.Metadata{
		Total:    int64(total),
		Filtered: int64(total),
		Limit:    limit,
		Offset:   offset,
	}

	h.sendFormattedResponse(w, modelPtr, metadata, options)
}

func (h *Handler) handleCreate(ctx context.Context, w common.ResponseWriter, data interface{}, options ExtendedRequestOptions) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleCreate", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)
	model := GetModel(ctx)

	logger.Info("Creating record in %s.%s", schema, entity)

	// Handle batch creation
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() == reflect.Slice || dataValue.Kind() == reflect.Array {
		logger.Debug("Batch creation detected, count: %d", dataValue.Len())

		// Use transaction for batch insert
		err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
			for i := 0; i < dataValue.Len(); i++ {
				item := dataValue.Index(i).Interface()

				// Convert item to model type - create a pointer to the model
				modelValue := reflect.New(reflect.TypeOf(model)).Interface()
				jsonData, err := json.Marshal(item)
				if err != nil {
					return fmt.Errorf("failed to marshal item: %w", err)
				}
				if err := json.Unmarshal(jsonData, modelValue); err != nil {
					return fmt.Errorf("failed to unmarshal item: %w", err)
				}

				query := tx.NewInsert().Model(modelValue).Table(tableName)
				if _, err := query.Exec(ctx); err != nil {
					return fmt.Errorf("failed to insert record: %w", err)
				}
			}
			return nil
		})

		if err != nil {
			logger.Error("Error creating records: %v", err)
			h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating records", err)
			return
		}

		h.sendResponse(w, map[string]interface{}{"created": dataValue.Len()}, nil)
		return
	}

	// Single record creation - create a pointer to the model
	modelValue := reflect.New(reflect.TypeOf(model)).Interface()
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error("Error marshaling data: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_data", "Invalid data format", err)
		return
	}
	if err := json.Unmarshal(jsonData, modelValue); err != nil {
		logger.Error("Error unmarshaling data: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_data", "Invalid data format", err)
		return
	}

	query := h.db.NewInsert().Model(modelValue).Table(tableName)
	if _, err := query.Exec(ctx); err != nil {
		logger.Error("Error creating record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating record", err)
		return
	}

	h.sendResponse(w, modelValue, nil)
}

func (h *Handler) handleUpdate(ctx context.Context, w common.ResponseWriter, id string, idPtr *int64, data interface{}, options ExtendedRequestOptions) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleUpdate", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)

	logger.Info("Updating record in %s.%s", schema, entity)

	// Convert data to map
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		jsonData, err := json.Marshal(data)
		if err != nil {
			logger.Error("Error marshaling data: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_data", "Invalid data format", err)
			return
		}
		if err := json.Unmarshal(jsonData, &dataMap); err != nil {
			logger.Error("Error unmarshaling data: %v", err)
			h.sendError(w, http.StatusBadRequest, "invalid_data", "Invalid data format", err)
			return
		}
	}

	query := h.db.NewUpdate().Table(tableName).SetMap(dataMap)

	// Apply ID filter
	if id != "" {
		query = query.Where("id = ?", id)
	} else if idPtr != nil {
		query = query.Where("id = ?", *idPtr)
	} else {
		h.sendError(w, http.StatusBadRequest, "missing_id", "ID is required for update", nil)
		return
	}

	result, err := query.Exec(ctx)
	if err != nil {
		logger.Error("Error updating record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "update_error", "Error updating record", err)
		return
	}

	h.sendResponse(w, map[string]interface{}{
		"updated": result.RowsAffected(),
	}, nil)
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

	logger.Info("Deleting record from %s.%s", schema, entity)

	query := h.db.NewDelete().Table(tableName)

	if id == "" {
		h.sendError(w, http.StatusBadRequest, "missing_id", "ID is required for delete", nil)
		return
	}

	query = query.Where("id = ?", id)

	result, err := query.Exec(ctx)
	if err != nil {
		logger.Error("Error deleting record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting record", err)
		return
	}

	h.sendResponse(w, map[string]interface{}{
		"deleted": result.RowsAffected(),
	}, nil)
}

func (h *Handler) applyFilter(query common.SelectQuery, filter common.FilterOption) common.SelectQuery {
	switch strings.ToLower(filter.Operator) {
	case "eq", "equals":
		return query.Where(fmt.Sprintf("%s = ?", filter.Column), filter.Value)
	case "neq", "not_equals", "ne":
		return query.Where(fmt.Sprintf("%s != ?", filter.Column), filter.Value)
	case "gt", "greater_than":
		return query.Where(fmt.Sprintf("%s > ?", filter.Column), filter.Value)
	case "gte", "greater_than_equals", "ge":
		return query.Where(fmt.Sprintf("%s >= ?", filter.Column), filter.Value)
	case "lt", "less_than":
		return query.Where(fmt.Sprintf("%s < ?", filter.Column), filter.Value)
	case "lte", "less_than_equals", "le":
		return query.Where(fmt.Sprintf("%s <= ?", filter.Column), filter.Value)
	case "like":
		return query.Where(fmt.Sprintf("%s LIKE ?", filter.Column), filter.Value)
	case "ilike":
		// Use ILIKE for case-insensitive search (PostgreSQL)
		// For other databases, cast to citext or use LOWER()
		return query.Where(fmt.Sprintf("CAST(%s AS TEXT) ILIKE ?", filter.Column), filter.Value)
	case "in":
		return query.Where(fmt.Sprintf("%s IN (?)", filter.Column), filter.Value)
	case "between":
		// Handle between operator - exclusive (> val1 AND < val2)
		if values, ok := filter.Value.([]interface{}); ok && len(values) == 2 {
			return query.Where(fmt.Sprintf("%s > ? AND %s < ?", filter.Column, filter.Column), values[0], values[1])
		} else if values, ok := filter.Value.([]string); ok && len(values) == 2 {
			return query.Where(fmt.Sprintf("%s > ? AND %s < ?", filter.Column, filter.Column), values[0], values[1])
		}
		logger.Warn("Invalid BETWEEN filter value format")
		return query
	case "between_inclusive":
		// Handle between inclusive operator - inclusive (>= val1 AND <= val2)
		if values, ok := filter.Value.([]interface{}); ok && len(values) == 2 {
			return query.Where(fmt.Sprintf("%s >= ? AND %s <= ?", filter.Column, filter.Column), values[0], values[1])
		} else if values, ok := filter.Value.([]string); ok && len(values) == 2 {
			return query.Where(fmt.Sprintf("%s >= ? AND %s <= ?", filter.Column, filter.Column), values[0], values[1])
		}
		logger.Warn("Invalid BETWEEN INCLUSIVE filter value format")
		return query
	case "is_null", "isnull":
		// Check for NULL values
		return query.Where(fmt.Sprintf("(%s IS NULL OR %s = '')", filter.Column, filter.Column))
	case "is_not_null", "isnotnull":
		// Check for NOT NULL values
		return query.Where(fmt.Sprintf("(%s IS NOT NULL AND %s != '')", filter.Column, filter.Column))
	default:
		logger.Warn("Unknown filter operator: %s, defaulting to equals", filter.Operator)
		return query.Where(fmt.Sprintf("%s = ?", filter.Column), filter.Value)
	}
}

// parseTableName splits a table name that may contain schema into separate schema and table
func (h *Handler) parseTableName(fullTableName string) (schema, table string) {
	if idx := strings.LastIndex(fullTableName, "."); idx != -1 {
		return fullTableName[:idx], fullTableName[idx+1:]
	}
	return "", fullTableName
}

// getSchemaAndTable returns the schema and table name separately
// It checks SchemaProvider and TableNameProvider interfaces and handles cases where
// the table name may already include the schema (e.g., "public.users")
//
// Priority order:
// 1. If TableName() contains a schema (e.g., "myschema.mytable"), that schema takes precedence
// 2. If model implements SchemaProvider, use that schema
// 3. Otherwise, use the defaultSchema parameter
func (h *Handler) getSchemaAndTable(defaultSchema, entity string, model interface{}) (schema, table string) {
	// First check if model provides a table name
	// We check this FIRST because the table name might already contain the schema
	if tableProvider, ok := model.(common.TableNameProvider); ok {
		tableName := tableProvider.TableName()

		// IMPORTANT: Check if the table name already contains a schema (e.g., "schema.table")
		// This is common when models need to specify a different schema than the default
		if tableSchema, tableOnly := h.parseTableName(tableName); tableSchema != "" {
			// Table name includes schema - use it and ignore any other schema providers
			logger.Debug("TableName() includes schema: %s.%s", tableSchema, tableOnly)
			return tableSchema, tableOnly
		}

		// Table name is just the table name without schema
		// Now determine which schema to use
		if schemaProvider, ok := model.(common.SchemaProvider); ok {
			schema = schemaProvider.SchemaName()
		} else {
			schema = defaultSchema
		}

		return schema, tableName
	}

	// No TableNameProvider, so check for schema and use entity as table name
	if schemaProvider, ok := model.(common.SchemaProvider); ok {
		schema = schemaProvider.SchemaName()
	} else {
		schema = defaultSchema
	}

	// Default to entity name as table
	return schema, entity
}

// getTableName returns the full table name including schema (schema.table)
func (h *Handler) getTableName(schema, entity string, model interface{}) string {
	schemaName, tableName := h.getSchemaAndTable(schema, entity, model)
	if schemaName != "" {
		return fmt.Sprintf("%s.%s", schemaName, tableName)
	}
	return tableName
}

func (h *Handler) generateMetadata(schema, entity string, model interface{}) *common.TableMetadata {
	modelType := reflect.TypeOf(model)

	// Unwrap pointers, slices, and arrays to get to the base struct type
	for modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array {
		modelType = modelType.Elem()
	}

	// Validate that we have a struct type
	if modelType.Kind() != reflect.Struct {
		logger.Error("Model type must be a struct, got %s for %s.%s", modelType.Kind(), schema, entity)
		return &common.TableMetadata{
			Schema:  schema,
			Table:   h.getTableName(schema, entity, model),
			Columns: []common.Column{},
		}
	}

	tableName := h.getTableName(schema, entity, model)

	metadata := &common.TableMetadata{
		Schema:  schema,
		Table:   tableName,
		Columns: []common.Column{},
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Get column name from gorm tag or json tag
		columnName := field.Tag.Get("gorm")
		if strings.Contains(columnName, "column:") {
			parts := strings.Split(columnName, ";")
			for _, part := range parts {
				if strings.HasPrefix(part, "column:") {
					columnName = strings.TrimPrefix(part, "column:")
					break
				}
			}
		} else {
			columnName = field.Tag.Get("json")
			if columnName == "" || columnName == "-" {
				columnName = strings.ToLower(field.Name)
			}
		}

		// Check for primary key and unique constraint
		gormTag := field.Tag.Get("gorm")

		column := common.Column{
			Name:       columnName,
			Type:       h.getColumnType(field.Type),
			IsNullable: h.isNullable(field),
			IsPrimary:  strings.Contains(gormTag, "primaryKey") || strings.Contains(gormTag, "primary_key"),
			IsUnique:   strings.Contains(gormTag, "unique"),
			HasIndex:   strings.Contains(gormTag, "index"),
		}

		metadata.Columns = append(metadata.Columns, column)
	}

	return metadata
}

func (h *Handler) getColumnType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "boolean"
	case reflect.Ptr:
		return h.getColumnType(t.Elem())
	default:
		return "unknown"
	}
}

func (h *Handler) isNullable(field reflect.StructField) bool {
	return field.Type.Kind() == reflect.Ptr
}

func (h *Handler) sendResponse(w common.ResponseWriter, data interface{}, metadata *common.Metadata) {
	response := common.Response{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	}
	w.WriteHeader(http.StatusOK)
	w.WriteJSON(response)
}

// sendFormattedResponse sends response with formatting options
func (h *Handler) sendFormattedResponse(w common.ResponseWriter, data interface{}, metadata *common.Metadata, options ExtendedRequestOptions) {
	// Clean JSON if requested (remove null/empty fields)
	if options.CleanJSON {
		data = h.cleanJSON(data)
	}
	w.SetHeader("Content-Type", "application/json")
	// Format response based on response format option
	switch options.ResponseFormat {
	case "simple":
		// Simple format: just return the data array
		w.WriteHeader(http.StatusOK)
		w.WriteJSON(data)
	case "syncfusion":
		// Syncfusion format: { result: data, count: total }
		response := map[string]interface{}{
			"result": data,
		}
		if metadata != nil {
			response["count"] = metadata.Total
		}
		w.WriteHeader(http.StatusOK)
		w.WriteJSON(response)
	default:
		// Default/detail format: standard response with metadata
		response := common.Response{
			Success:  true,
			Data:     data,
			Metadata: metadata,
		}
		w.WriteHeader(http.StatusOK)
		w.WriteJSON(response)
	}
}

// cleanJSON removes null and empty fields from the response
func (h *Handler) cleanJSON(data interface{}) interface{} {
	// This is a simplified implementation
	// A full implementation would recursively clean nested structures
	// For now, we'll return the data as-is
	// TODO: Implement recursive cleaning
	return data
}

func (h *Handler) sendError(w common.ResponseWriter, statusCode int, code, message string, err error) {
	var details string
	if err != nil {
		details = err.Error()
	}

	response := common.Response{
		Success: false,
		Error: &common.APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	w.WriteHeader(statusCode)
	w.WriteJSON(response)
}

// filterExtendedOptions filters all column references, removing invalid ones and logging warnings
func filterExtendedOptions(validator *common.ColumnValidator, options ExtendedRequestOptions) ExtendedRequestOptions {
	filtered := options

	// Filter base RequestOptions
	filtered.RequestOptions = validator.FilterRequestOptions(options.RequestOptions)

	// Filter SearchColumns
	filtered.SearchColumns = validator.FilterValidColumns(options.SearchColumns)

	// Filter AdvancedSQL column keys
	filteredAdvSQL := make(map[string]string)
	for colName, sqlExpr := range options.AdvancedSQL {
		if validator.IsValidColumn(colName) {
			filteredAdvSQL[colName] = sqlExpr
		} else {
			logger.Warn("Invalid column in advanced SQL removed: %s", colName)
		}
	}
	filtered.AdvancedSQL = filteredAdvSQL

	// ComputedQL columns are allowed to be any name since they're computed
	// No filtering needed for ComputedQL keys
	filtered.ComputedQL = options.ComputedQL

	// Filter Expand columns
	filteredExpands := make([]ExpandOption, 0, len(options.Expand))
	for _, expand := range options.Expand {
		filteredExpand := expand
		// Don't validate relation name, only columns
		filteredExpand.Columns = validator.FilterValidColumns(expand.Columns)
		filteredExpands = append(filteredExpands, filteredExpand)
	}
	filtered.Expand = filteredExpands

	return filtered
}
