package restheadspec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/bitechdev/ResolveSpec/pkg/common"
	"github.com/bitechdev/ResolveSpec/pkg/logger"
	"github.com/bitechdev/ResolveSpec/pkg/reflection"
)

// Handler handles API requests using database and model abstractions
// This handler reads filters, columns, and options from HTTP headers
type Handler struct {
	db              common.Database
	registry        common.ModelRegistry
	hooks           *HookRegistry
	nestedProcessor *common.NestedCUDProcessor
}

// NewHandler creates a new API handler with database and registry abstractions
func NewHandler(db common.Database, registry common.ModelRegistry) *Handler {
	handler := &Handler{
		db:       db,
		registry: registry,
		hooks:    NewHookRegistry(),
	}
	// Initialize nested processor
	handler.nestedProcessor = common.NewNestedCUDProcessor(db, registry, handler)
	return handler
}

// Hooks returns the hook registry for this handler
// Use this to register custom hooks for operations
func (h *Handler) Hooks() *HookRegistry {
	return h.hooks
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
		// Try to read body for batch delete support
		var data interface{}
		body, err := r.Body()
		if err == nil && len(body) > 0 {
			if err := json.Unmarshal(body, &data); err != nil {
				logger.Warn("Failed to decode delete request body (will try single delete): %v", err)
				data = nil
			}
		}
		h.handleDelete(ctx, w, id, data)
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

	// Execute BeforeRead hooks
	hookCtx := &HookContext{
		Context:   ctx,
		Handler:   h,
		Schema:    schema,
		Entity:    entity,
		TableName: tableName,
		Model:     model,
		Options:   options,
		ID:        id,
		Writer:    w,
	}

	if err := h.hooks.Execute(BeforeRead, hookCtx); err != nil {
		logger.Error("BeforeRead hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

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

	// Start with Model() using the slice pointer to avoid "Model(nil)" errors in Count()
	// Bun's Model() accepts both single pointers and slice pointers
	query := h.db.NewSelect().Model(modelPtr)

	// Only set Table() if the model doesn't provide a table name via the underlying type
	// Create a temporary instance to check for TableNameProvider
	tempInstance := reflect.New(modelType).Interface()
	if provider, ok := tempInstance.(common.TableNameProvider); !ok || provider.TableName() == "" {
		query = query.Table(tableName)
	}

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

	// Apply filters - validate and adjust for column types first
	for i := range options.Filters {
		filter := &options.Filters[i]

		// Validate and adjust filter based on column type
		castInfo := h.ValidateAndAdjustFilterForColumnType(filter, model)

		// Default to AND if LogicOperator is not set
		logicOp := filter.LogicOperator
		if logicOp == "" {
			logicOp = "AND"
		}

		logger.Debug("Applying filter: %s %s %v (needsCast=%v, logic=%s)", filter.Column, filter.Operator, filter.Value, castInfo.NeedsCast, logicOp)
		query = h.applyFilter(query, *filter, tableName, castInfo.NeedsCast, logicOp)
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
		if strings.EqualFold(sort.Direction, "desc") {
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

	// Apply cursor-based pagination
	if len(options.CursorForward) > 0 || len(options.CursorBackward) > 0 {
		logger.Debug("Applying cursor pagination")

		// Get primary key name
		pkName := reflection.GetPrimaryKeyName(model)

		// Extract model columns for validation using the generic database function
		modelColumns := reflection.GetModelColumns(model)

		// Build expand joins map (if needed in future)
		var expandJoins map[string]string
		if len(options.Expand) > 0 {
			expandJoins = make(map[string]string)
			// TODO: Build actual JOIN SQL for each expand relation
			// For now, pass empty map as joins are handled via Preload
		}

		// Get cursor filter SQL
		cursorFilter, err := options.GetCursorFilter(tableName, pkName, modelColumns, expandJoins)
		if err != nil {
			logger.Error("Error building cursor filter: %v", err)
			h.sendError(w, http.StatusBadRequest, "cursor_error", "Invalid cursor pagination", err)
			return
		}

		// Apply cursor filter to query
		if cursorFilter != "" {
			logger.Debug("Applying cursor filter: %s", cursorFilter)
			query = query.Where(cursorFilter)
		}
	}

	// Execute BeforeScan hooks - pass query chain so hooks can modify it
	hookCtx.Query = query
	if err := h.hooks.Execute(BeforeScan, hookCtx); err != nil {
		logger.Error("BeforeScan hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	// Use potentially modified query from hook context
	if modifiedQuery, ok := hookCtx.Query.(common.SelectQuery); ok {
		query = modifiedQuery
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

	// Set row numbers on each record if the model has a RowNumber field
	h.setRowNumbersOnRecords(modelPtr, offset)

	metadata := &common.Metadata{
		Total:    int64(total),
		Count:    int64(common.Len(modelPtr)),
		Filtered: int64(total),
		Limit:    limit,
		Offset:   offset,
	}

	// Fetch row number for a specific record if requested
	if options.FetchRowNumber != nil && *options.FetchRowNumber != "" {
		pkName := reflection.GetPrimaryKeyName(model)
		pkValue := *options.FetchRowNumber

		logger.Debug("Fetching row number for specific PK %s = %s", pkName, pkValue)

		rowNum, err := h.FetchRowNumber(ctx, tableName, pkName, pkValue, options, model)
		if err != nil {
			logger.Warn("Failed to fetch row number: %v", err)
			// Don't fail the entire request, just log the warning
		} else {
			metadata.RowNumber = &rowNum
			logger.Debug("Row number for PK %s: %d", pkValue, rowNum)
		}
	}

	// Execute AfterRead hooks
	hookCtx.Result = modelPtr
	hookCtx.Error = nil

	if err := h.hooks.Execute(AfterRead, hookCtx); err != nil {
		logger.Error("AfterRead hook failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "hook_error", "Hook execution failed", err)
		return
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

	// Check if data is a single map with nested relations
	if dataMap, ok := data.(map[string]interface{}); ok {
		if h.shouldUseNestedProcessor(dataMap, model) {
			logger.Info("Using nested CUD processor for create operation")
			result, err := h.nestedProcessor.ProcessNestedCUD(ctx, "insert", dataMap, model, make(map[string]interface{}), tableName)
			if err != nil {
				logger.Error("Error in nested create: %v", err)
				h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating record with nested data", err)
				return
			}
			logger.Info("Successfully created record with nested data, ID: %v", result.ID)
			h.sendResponse(w, result.Data, nil)
			return
		}
	}

	// Execute BeforeCreate hooks
	hookCtx := &HookContext{
		Context:   ctx,
		Handler:   h,
		Schema:    schema,
		Entity:    entity,
		TableName: tableName,
		Model:     model,
		Options:   options,
		Data:      data,
		Writer:    w,
	}

	if err := h.hooks.Execute(BeforeCreate, hookCtx); err != nil {
		logger.Error("BeforeCreate hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	// Use potentially modified data from hook context
	data = hookCtx.Data

	// Handle batch creation
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() == reflect.Slice || dataValue.Kind() == reflect.Array {
		logger.Debug("Batch creation detected, count: %d", dataValue.Len())

		// Check if any item needs nested processing
		hasNestedData := false
		for i := 0; i < dataValue.Len(); i++ {
			item := dataValue.Index(i).Interface()
			if itemMap, ok := item.(map[string]interface{}); ok {
				if h.shouldUseNestedProcessor(itemMap, model) {
					hasNestedData = true
					break
				}
			}
		}

		if hasNestedData {
			logger.Info("Using nested CUD processor for batch create with nested data")
			results := make([]interface{}, 0, dataValue.Len())
			err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
				// Temporarily swap the database to use transaction
				originalDB := h.nestedProcessor
				h.nestedProcessor = common.NewNestedCUDProcessor(tx, h.registry, h)
				defer func() {
					h.nestedProcessor = originalDB
				}()

				for i := 0; i < dataValue.Len(); i++ {
					item := dataValue.Index(i).Interface()
					if itemMap, ok := item.(map[string]interface{}); ok {
						result, err := h.nestedProcessor.ProcessNestedCUD(ctx, "insert", itemMap, model, make(map[string]interface{}), tableName)
						if err != nil {
							return fmt.Errorf("failed to process item: %w", err)
						}
						results = append(results, result.Data)
					}
				}
				return nil
			})
			if err != nil {
				logger.Error("Error creating records with nested data: %v", err)
				h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating records with nested data", err)
				return
			}

			// Execute AfterCreate hooks
			hookCtx.Result = map[string]interface{}{"created": len(results), "data": results}
			hookCtx.Error = nil

			if err := h.hooks.Execute(AfterCreate, hookCtx); err != nil {
				logger.Error("AfterCreate hook failed: %v", err)
				h.sendError(w, http.StatusInternalServerError, "hook_error", "Hook execution failed", err)
				return
			}

			logger.Info("Successfully created %d records with nested data", len(results))
			h.sendResponse(w, results, nil)
			return
		}

		// Standard batch insert without nested relations
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

				// Execute BeforeScan hooks - pass query chain so hooks can modify it
				batchHookCtx := &HookContext{
					Context:   ctx,
					Handler:   h,
					Schema:    schema,
					Entity:    entity,
					TableName: tableName,
					Model:     model,
					Options:   options,
					Data:      modelValue,
					Writer:    w,
					Query:     query,
				}
				if err := h.hooks.Execute(BeforeScan, batchHookCtx); err != nil {
					return fmt.Errorf("BeforeScan hook failed: %w", err)
				}

				// Use potentially modified query from hook context
				if modifiedQuery, ok := batchHookCtx.Query.(common.InsertQuery); ok {
					query = modifiedQuery
				}

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

		// Execute AfterCreate hooks for batch creation
		hookCtx.Result = map[string]interface{}{"created": dataValue.Len()}
		hookCtx.Error = nil

		if err := h.hooks.Execute(AfterCreate, hookCtx); err != nil {
			logger.Error("AfterCreate hook failed: %v", err)
			h.sendError(w, http.StatusInternalServerError, "hook_error", "Hook execution failed", err)
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

	// Execute BeforeScan hooks - pass query chain so hooks can modify it
	hookCtx.Data = modelValue
	hookCtx.Query = query
	if err := h.hooks.Execute(BeforeScan, hookCtx); err != nil {
		logger.Error("BeforeScan hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	// Use potentially modified query from hook context
	if modifiedQuery, ok := hookCtx.Query.(common.InsertQuery); ok {
		query = modifiedQuery
	}

	if _, err := query.Exec(ctx); err != nil {
		logger.Error("Error creating record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "create_error", "Error creating record", err)
		return
	}

	// Execute AfterCreate hooks for single record creation
	hookCtx.Result = modelValue
	hookCtx.Error = nil

	if err := h.hooks.Execute(AfterCreate, hookCtx); err != nil {
		logger.Error("AfterCreate hook failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "hook_error", "Hook execution failed", err)
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
	model := GetModel(ctx)

	logger.Info("Updating record in %s.%s", schema, entity)

	// Convert data to map first for nested processor check
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

	// Check if we should use nested processing
	if h.shouldUseNestedProcessor(dataMap, model) {
		logger.Info("Using nested CUD processor for update operation")
		// Ensure ID is in the data map
		var targetID interface{}
		if id != "" {
			targetID = id
		} else if idPtr != nil {
			targetID = *idPtr
		}
		if targetID != nil {
			dataMap["id"] = targetID
		}
		result, err := h.nestedProcessor.ProcessNestedCUD(ctx, "update", dataMap, model, make(map[string]interface{}), tableName)
		if err != nil {
			logger.Error("Error in nested update: %v", err)
			h.sendError(w, http.StatusInternalServerError, "update_error", "Error updating record with nested data", err)
			return
		}
		logger.Info("Successfully updated record with nested data, rows: %d", result.AffectedRows)
		h.sendResponse(w, result.Data, nil)
		return
	}

	// Execute BeforeUpdate hooks
	hookCtx := &HookContext{
		Context:   ctx,
		Handler:   h,
		Schema:    schema,
		Entity:    entity,
		TableName: tableName,
		Model:     model,
		Options:   options,
		ID:        id,
		Data:      data,
		Writer:    w,
	}

	if err := h.hooks.Execute(BeforeUpdate, hookCtx); err != nil {
		logger.Error("BeforeUpdate hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	// Use potentially modified data from hook context
	data = hookCtx.Data

	// Convert data to map (again if modified by hooks)
	dataMap, ok = data.(map[string]interface{})
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
	switch {
	case id != "":
		query = query.Where("id = ?", id)
	case idPtr != nil:
		query = query.Where("id = ?", *idPtr)
	default:
		h.sendError(w, http.StatusBadRequest, "missing_id", "ID is required for update", nil)
		return
	}

	// Execute BeforeScan hooks - pass query chain so hooks can modify it
	hookCtx.Query = query
	if err := h.hooks.Execute(BeforeScan, hookCtx); err != nil {
		logger.Error("BeforeScan hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	// Use potentially modified query from hook context
	if modifiedQuery, ok := hookCtx.Query.(common.UpdateQuery); ok {
		query = modifiedQuery
	}

	result, err := query.Exec(ctx)
	if err != nil {
		logger.Error("Error updating record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "update_error", "Error updating record", err)
		return
	}

	// Execute AfterUpdate hooks
	responseData := map[string]interface{}{
		"updated": result.RowsAffected(),
	}
	hookCtx.Result = responseData
	hookCtx.Error = nil

	if err := h.hooks.Execute(AfterUpdate, hookCtx); err != nil {
		logger.Error("AfterUpdate hook failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "hook_error", "Hook execution failed", err)
		return
	}

	h.sendResponse(w, responseData, nil)
}

func (h *Handler) handleDelete(ctx context.Context, w common.ResponseWriter, id string, data interface{}) {
	// Capture panics and return error response
	defer func() {
		if err := recover(); err != nil {
			h.handlePanic(w, "handleDelete", err)
		}
	}()

	schema := GetSchema(ctx)
	entity := GetEntity(ctx)
	tableName := GetTableName(ctx)
	model := GetModel(ctx)

	logger.Info("Deleting record(s) from %s.%s", schema, entity)

	// Handle batch delete from request data
	if data != nil {
		switch v := data.(type) {
		case []string:
			// Array of IDs as strings
			logger.Info("Batch delete with %d IDs ([]string)", len(v))
			deletedCount := 0
			err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
				for _, itemID := range v {
					// Execute hooks for each item
					hookCtx := &HookContext{
						Context:   ctx,
						Handler:   h,
						Schema:    schema,
						Entity:    entity,
						TableName: tableName,
						Model:     model,
						ID:        itemID,
						Writer:    w,
					}

					if err := h.hooks.Execute(BeforeDelete, hookCtx); err != nil {
						logger.Warn("BeforeDelete hook failed for ID %s: %v", itemID, err)
						continue
					}

					query := tx.NewDelete().Table(tableName).Where("id = ?", itemID)

					result, err := query.Exec(ctx)
					if err != nil {
						return fmt.Errorf("failed to delete record %s: %w", itemID, err)
					}
					deletedCount += int(result.RowsAffected())

					// Execute AfterDelete hook
					hookCtx.Result = map[string]interface{}{"deleted": result.RowsAffected()}
					hookCtx.Error = nil
					if err := h.hooks.Execute(AfterDelete, hookCtx); err != nil {
						logger.Warn("AfterDelete hook failed for ID %s: %v", itemID, err)
					}
				}
				return nil
			})
			if err != nil {
				logger.Error("Error in batch delete: %v", err)
				h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting records", err)
				return
			}
			logger.Info("Successfully deleted %d records", deletedCount)
			h.sendResponse(w, map[string]interface{}{"deleted": deletedCount}, nil)
			return

		case []interface{}:
			// Array of IDs or objects with ID field
			logger.Info("Batch delete with %d items ([]interface{})", len(v))
			deletedCount := 0
			err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
				for _, item := range v {
					var itemID interface{}

					// Check if item is a string ID or object with id field
					switch v := item.(type) {
					case string:
						itemID = v
					case map[string]interface{}:
						itemID = v["id"]
					default:
						itemID = item
					}

					if itemID == nil {
						continue
					}

					itemIDStr := fmt.Sprintf("%v", itemID)

					// Execute hooks for each item
					hookCtx := &HookContext{
						Context:   ctx,
						Handler:   h,
						Schema:    schema,
						Entity:    entity,
						TableName: tableName,
						Model:     model,
						ID:        itemIDStr,
						Writer:    w,
					}

					if err := h.hooks.Execute(BeforeDelete, hookCtx); err != nil {
						logger.Warn("BeforeDelete hook failed for ID %v: %v", itemID, err)
						continue
					}

					query := tx.NewDelete().Table(tableName).Where("id = ?", itemID)
					result, err := query.Exec(ctx)
					if err != nil {
						return fmt.Errorf("failed to delete record %v: %w", itemID, err)
					}
					deletedCount += int(result.RowsAffected())

					// Execute AfterDelete hook
					hookCtx.Result = map[string]interface{}{"deleted": result.RowsAffected()}
					hookCtx.Error = nil
					if err := h.hooks.Execute(AfterDelete, hookCtx); err != nil {
						logger.Warn("AfterDelete hook failed for ID %v: %v", itemID, err)
					}
				}
				return nil
			})
			if err != nil {
				logger.Error("Error in batch delete: %v", err)
				h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting records", err)
				return
			}
			logger.Info("Successfully deleted %d records", deletedCount)
			h.sendResponse(w, map[string]interface{}{"deleted": deletedCount}, nil)
			return

		case []map[string]interface{}:
			// Array of objects with id field
			logger.Info("Batch delete with %d items ([]map[string]interface{})", len(v))
			deletedCount := 0
			err := h.db.RunInTransaction(ctx, func(tx common.Database) error {
				for _, item := range v {
					if itemID, ok := item["id"]; ok && itemID != nil {
						itemIDStr := fmt.Sprintf("%v", itemID)

						// Execute hooks for each item
						hookCtx := &HookContext{
							Context:   ctx,
							Handler:   h,
							Schema:    schema,
							Entity:    entity,
							TableName: tableName,
							Model:     model,
							ID:        itemIDStr,
							Writer:    w,
						}

						if err := h.hooks.Execute(BeforeDelete, hookCtx); err != nil {
							logger.Warn("BeforeDelete hook failed for ID %v: %v", itemID, err)
							continue
						}

						query := tx.NewDelete().Table(tableName).Where("id = ?", itemID)
						result, err := query.Exec(ctx)
						if err != nil {
							return fmt.Errorf("failed to delete record %v: %w", itemID, err)
						}
						deletedCount += int(result.RowsAffected())

						// Execute AfterDelete hook
						hookCtx.Result = map[string]interface{}{"deleted": result.RowsAffected()}
						hookCtx.Error = nil
						if err := h.hooks.Execute(AfterDelete, hookCtx); err != nil {
							logger.Warn("AfterDelete hook failed for ID %v: %v", itemID, err)
						}
					}
				}
				return nil
			})
			if err != nil {
				logger.Error("Error in batch delete: %v", err)
				h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting records", err)
				return
			}
			logger.Info("Successfully deleted %d records", deletedCount)
			h.sendResponse(w, map[string]interface{}{"deleted": deletedCount}, nil)
			return

		case map[string]interface{}:
			// Single object with id field
			if itemID, ok := v["id"]; ok && itemID != nil {
				id = fmt.Sprintf("%v", itemID)
			}
		}
	}

	// Single delete with URL ID
	// Execute BeforeDelete hooks
	hookCtx := &HookContext{
		Context:   ctx,
		Handler:   h,
		Schema:    schema,
		Entity:    entity,
		TableName: tableName,
		Model:     model,
		ID:        id,
		Writer:    w,
	}

	if err := h.hooks.Execute(BeforeDelete, hookCtx); err != nil {
		logger.Error("BeforeDelete hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	query := h.db.NewDelete().Table(tableName)

	if id == "" {
		h.sendError(w, http.StatusBadRequest, "missing_id", "ID is required for delete", nil)
		return
	}

	query = query.Where("id = ?", id)

	// Execute BeforeScan hooks - pass query chain so hooks can modify it
	hookCtx.Query = query
	if err := h.hooks.Execute(BeforeScan, hookCtx); err != nil {
		logger.Error("BeforeScan hook failed: %v", err)
		h.sendError(w, http.StatusBadRequest, "hook_error", "Hook execution failed", err)
		return
	}

	// Use potentially modified query from hook context
	if modifiedQuery, ok := hookCtx.Query.(common.DeleteQuery); ok {
		query = modifiedQuery
	}

	result, err := query.Exec(ctx)
	if err != nil {
		logger.Error("Error deleting record: %v", err)
		h.sendError(w, http.StatusInternalServerError, "delete_error", "Error deleting record", err)
		return
	}

	// Execute AfterDelete hooks
	responseData := map[string]interface{}{
		"deleted": result.RowsAffected(),
	}
	hookCtx.Result = responseData
	hookCtx.Error = nil

	if err := h.hooks.Execute(AfterDelete, hookCtx); err != nil {
		logger.Error("AfterDelete hook failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "hook_error", "Hook execution failed", err)
		return
	}

	h.sendResponse(w, responseData, nil)
}

// qualifyColumnName ensures column name is fully qualified with table name if not already
func (h *Handler) qualifyColumnName(columnName, fullTableName string) string {
	// Check if column already has a table/schema prefix (contains a dot)
	if strings.Contains(columnName, ".") {
		return columnName
	}

	// If no table name provided, return column as-is
	if fullTableName == "" {
		return columnName
	}

	// Extract just the table name from "schema.table" format
	// Only use the table name part, not the schema
	tableOnly := fullTableName
	if idx := strings.LastIndex(fullTableName, "."); idx != -1 {
		tableOnly = fullTableName[idx+1:]
	}

	// Return column qualified with just the table name
	return fmt.Sprintf("%s.%s", tableOnly, columnName)
}

func (h *Handler) applyFilter(query common.SelectQuery, filter common.FilterOption, tableName string, needsCast bool, logicOp string) common.SelectQuery {
	// Qualify the column name with table name if not already qualified
	qualifiedColumn := h.qualifyColumnName(filter.Column, tableName)

	// Apply casting to text if needed for non-numeric columns or non-numeric values
	if needsCast {
		qualifiedColumn = fmt.Sprintf("CAST(%s AS TEXT)", qualifiedColumn)
	}

	// Helper function to apply the correct Where method based on logic operator
	applyWhere := func(condition string, args ...interface{}) common.SelectQuery {
		if logicOp == "OR" {
			return query.WhereOr(condition, args...)
		}
		return query.Where(condition, args...)
	}

	switch strings.ToLower(filter.Operator) {
	case "eq", "equals":
		return applyWhere(fmt.Sprintf("%s = ?", qualifiedColumn), filter.Value)
	case "neq", "not_equals", "ne":
		return applyWhere(fmt.Sprintf("%s != ?", qualifiedColumn), filter.Value)
	case "gt", "greater_than":
		return applyWhere(fmt.Sprintf("%s > ?", qualifiedColumn), filter.Value)
	case "gte", "greater_than_equals", "ge":
		return applyWhere(fmt.Sprintf("%s >= ?", qualifiedColumn), filter.Value)
	case "lt", "less_than":
		return applyWhere(fmt.Sprintf("%s < ?", qualifiedColumn), filter.Value)
	case "lte", "less_than_equals", "le":
		return applyWhere(fmt.Sprintf("%s <= ?", qualifiedColumn), filter.Value)
	case "like":
		return applyWhere(fmt.Sprintf("%s LIKE ?", qualifiedColumn), filter.Value)
	case "ilike":
		// Use ILIKE for case-insensitive search (PostgreSQL)
		// Column is already cast to TEXT if needed
		return applyWhere(fmt.Sprintf("%s ILIKE ?", qualifiedColumn), filter.Value)
	case "in":
		return applyWhere(fmt.Sprintf("%s IN (?)", qualifiedColumn), filter.Value)
	case "between":
		// Handle between operator - exclusive (> val1 AND < val2)
		if values, ok := filter.Value.([]interface{}); ok && len(values) == 2 {
			return applyWhere(fmt.Sprintf("%s > ? AND %s < ?", qualifiedColumn, qualifiedColumn), values[0], values[1])
		} else if values, ok := filter.Value.([]string); ok && len(values) == 2 {
			return applyWhere(fmt.Sprintf("%s > ? AND %s < ?", qualifiedColumn, qualifiedColumn), values[0], values[1])
		}
		logger.Warn("Invalid BETWEEN filter value format")
		return query
	case "between_inclusive":
		// Handle between inclusive operator - inclusive (>= val1 AND <= val2)
		if values, ok := filter.Value.([]interface{}); ok && len(values) == 2 {
			return applyWhere(fmt.Sprintf("%s >= ? AND %s <= ?", qualifiedColumn, qualifiedColumn), values[0], values[1])
		} else if values, ok := filter.Value.([]string); ok && len(values) == 2 {
			return applyWhere(fmt.Sprintf("%s >= ? AND %s <= ?", qualifiedColumn, qualifiedColumn), values[0], values[1])
		}
		logger.Warn("Invalid BETWEEN INCLUSIVE filter value format")
		return query
	case "is_null", "isnull":
		// Check for NULL values - don't use cast for NULL checks
		colName := h.qualifyColumnName(filter.Column, tableName)
		return applyWhere(fmt.Sprintf("(%s IS NULL OR %s = '')", colName, colName))
	case "is_not_null", "isnotnull":
		// Check for NOT NULL values - don't use cast for NULL checks
		colName := h.qualifyColumnName(filter.Column, tableName)
		return applyWhere(fmt.Sprintf("(%s IS NOT NULL AND %s != '')", colName, colName))
	default:
		logger.Warn("Unknown filter operator: %s, defaulting to equals", filter.Operator)
		return applyWhere(fmt.Sprintf("%s = ?", qualifiedColumn), filter.Value)
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
	if err := w.WriteJSON(response); err != nil {
		logger.Error("Failed to write JSON response: %v", err)
	}
}

// sendFormattedResponse sends response with formatting options
func (h *Handler) sendFormattedResponse(w common.ResponseWriter, data interface{}, metadata *common.Metadata, options ExtendedRequestOptions) {
	// Clean JSON if requested (remove null/empty fields)
	if options.CleanJSON {
		data = h.cleanJSON(data)
	}

	w.SetHeader("Content-Type", "application/json")
	w.SetHeader("Content-Range", fmt.Sprintf("%d-%d/%d", metadata.Offset, int64(metadata.Offset)+metadata.Count, metadata.Filtered))
	w.SetHeader("X-Api-Range-Total", fmt.Sprintf("%d", metadata.Filtered))
	w.SetHeader("X-Api-Range-Size", fmt.Sprintf("%d", metadata.Count))

	// Format response based on response format option
	switch options.ResponseFormat {
	case "simple":
		// Simple format: just return the data array
		w.WriteHeader(http.StatusOK)
		if err := w.WriteJSON(data); err != nil {
			logger.Error("Failed to write JSON response: %v", err)
		}
	case "syncfusion":
		// Syncfusion format: { result: data, count: total }
		response := map[string]interface{}{
			"result": data,
		}
		if metadata != nil {
			response["count"] = metadata.Total
		}
		w.WriteHeader(http.StatusOK)
		if err := w.WriteJSON(response); err != nil {
			logger.Error("Failed to write JSON response: %v", err)
		}
	default:
		// Default/detail format: standard response with metadata
		response := common.Response{
			Success:  true,
			Data:     data,
			Metadata: metadata,
		}
		w.WriteHeader(http.StatusOK)
		if err := w.WriteJSON(response); err != nil {
			logger.Error("Failed to write JSON response: %v", err)
		}
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
	if err := w.WriteJSON(response); err != nil {
		logger.Error("Failed to write JSON error response: %v", err)
	}
}

// FetchRowNumber calculates the row number of a specific record based on sorting and filtering
// Returns the 1-based row number of the record with the given primary key value
func (h *Handler) FetchRowNumber(ctx context.Context, tableName string, pkName string, pkValue string, options ExtendedRequestOptions, model any) (int64, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic during FetchRowNumber: %v", r)
		}
	}()

	// Build the sort order SQL
	sortSQL := ""
	if len(options.Sort) > 0 {
		sortParts := make([]string, 0, len(options.Sort))
		for _, sort := range options.Sort {
			direction := "ASC"
			if strings.EqualFold(sort.Direction, "desc") {
				direction = "DESC"
			}
			sortParts = append(sortParts, fmt.Sprintf("%s.%s %s", tableName, sort.Column, direction))
		}
		sortSQL = strings.Join(sortParts, ", ")
	} else {
		// Default sort by primary key
		sortSQL = fmt.Sprintf("%s.%s ASC", tableName, pkName)
	}

	// Build WHERE clauses from filters
	whereClauses := make([]string, 0)
	for i := range options.Filters {
		filter := &options.Filters[i]
		whereClause := h.buildFilterSQL(filter, tableName)
		if whereClause != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("(%s)", whereClause))
		}
	}

	// Combine WHERE clauses
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add custom SQL WHERE if provided
	if options.CustomSQLWhere != "" {
		if whereSQL == "" {
			whereSQL = "WHERE " + options.CustomSQLWhere
		} else {
			whereSQL += " AND (" + options.CustomSQLWhere + ")"
		}
	}

	// Build JOIN clauses from Expand options
	joinSQL := ""
	if len(options.Expand) > 0 {
		joinParts := make([]string, 0, len(options.Expand))
		for _, expand := range options.Expand {
			// Note: This is a simplified join - in production you'd need proper FK mapping
			joinParts = append(joinParts, fmt.Sprintf("LEFT JOIN %s ON %s.%s_id = %s.id",
				expand.Relation, tableName, expand.Relation, expand.Relation))
		}
		joinSQL = strings.Join(joinParts, "\n")
	}

	// Build the final query with parameterized PK value
	queryStr := fmt.Sprintf(`
		SELECT search.rn
		FROM (
			SELECT %[1]s.%[2]s,
				ROW_NUMBER() OVER(ORDER BY %[3]s) AS rn
			FROM %[1]s
			%[5]s
			%[4]s
		) search
		WHERE search.%[2]s = ?
	`,
		tableName, // [1] - table name
		pkName,    // [2] - primary key column name
		sortSQL,   // [3] - sort order SQL
		whereSQL,  // [4] - WHERE clause
		joinSQL,   // [5] - JOIN clauses
	)

	logger.Debug("FetchRowNumber query: %s, pkValue: %s", queryStr, pkValue)

	// Execute the raw query with parameterized PK value
	var result []struct {
		RN int64 `bun:"rn"`
	}
	err := h.db.Query(ctx, &result, queryStr, pkValue)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch row number: %w", err)
	}

	if len(result) == 0 {
		return 0, fmt.Errorf("no row found for primary key %s", pkValue)
	}

	return result[0].RN, nil
}

// buildFilterSQL converts a filter to SQL WHERE clause string
func (h *Handler) buildFilterSQL(filter *common.FilterOption, tableName string) string {
	qualifiedColumn := h.qualifyColumnName(filter.Column, tableName)

	switch strings.ToLower(filter.Operator) {
	case "eq", "equals":
		return fmt.Sprintf("%s = '%v'", qualifiedColumn, filter.Value)
	case "neq", "not_equals", "ne":
		return fmt.Sprintf("%s != '%v'", qualifiedColumn, filter.Value)
	case "gt", "greater_than":
		return fmt.Sprintf("%s > '%v'", qualifiedColumn, filter.Value)
	case "gte", "greater_than_equals", "ge":
		return fmt.Sprintf("%s >= '%v'", qualifiedColumn, filter.Value)
	case "lt", "less_than":
		return fmt.Sprintf("%s < '%v'", qualifiedColumn, filter.Value)
	case "lte", "less_than_equals", "le":
		return fmt.Sprintf("%s <= '%v'", qualifiedColumn, filter.Value)
	case "like":
		return fmt.Sprintf("%s LIKE '%v'", qualifiedColumn, filter.Value)
	case "ilike":
		return fmt.Sprintf("%s ILIKE '%v'", qualifiedColumn, filter.Value)
	case "in":
		if values, ok := filter.Value.([]any); ok {
			valueStrs := make([]string, len(values))
			for i, v := range values {
				valueStrs[i] = fmt.Sprintf("'%v'", v)
			}
			return fmt.Sprintf("%s IN (%s)", qualifiedColumn, strings.Join(valueStrs, ", "))
		}
		return ""
	case "is_null", "isnull":
		return fmt.Sprintf("(%s IS NULL OR %s = '')", qualifiedColumn, qualifiedColumn)
	case "is_not_null", "isnotnull":
		return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", qualifiedColumn, qualifiedColumn)
	default:
		logger.Warn("Unknown filter operator in buildFilterSQL: %s", filter.Operator)
		return ""
	}
}

// setRowNumbersOnRecords sets the RowNumber field on each record if it exists
// The row number is calculated as offset + index + 1 (1-based)
func (h *Handler) setRowNumbersOnRecords(records any, offset int) {
	// Get the reflect value of the records
	recordsValue := reflect.ValueOf(records)
	if recordsValue.Kind() == reflect.Ptr {
		recordsValue = recordsValue.Elem()
	}

	// Ensure it's a slice
	if recordsValue.Kind() != reflect.Slice {
		logger.Debug("setRowNumbersOnRecords: records is not a slice, skipping")
		return
	}

	// Iterate through each record
	for i := 0; i < recordsValue.Len(); i++ {
		record := recordsValue.Index(i)

		// Dereference if it's a pointer
		if record.Kind() == reflect.Ptr {
			if record.IsNil() {
				continue
			}
			record = record.Elem()
		}

		// Ensure it's a struct
		if record.Kind() != reflect.Struct {
			continue
		}

		// Try to find and set the RowNumber field
		rowNumberField := record.FieldByName("RowNumber")
		if rowNumberField.IsValid() && rowNumberField.CanSet() {
			// Check if the field is of type int64
			if rowNumberField.Kind() == reflect.Int64 {
				rowNum := int64(offset + i + 1)
				rowNumberField.SetInt(rowNum)

			}
		}
	}
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

// shouldUseNestedProcessor determines if we should use nested CUD processing
// It checks if the data contains nested relations or a _request field
func (h *Handler) shouldUseNestedProcessor(data map[string]interface{}, model interface{}) bool {
	return common.ShouldUseNestedProcessor(data, model, h)
}

// Relationship support functions for nested CUD processing

// GetRelationshipInfo implements common.RelationshipInfoProvider interface
func (h *Handler) GetRelationshipInfo(modelType reflect.Type, relationName string) *common.RelationshipInfo {
	info := h.getRelationshipInfo(modelType, relationName)
	if info == nil {
		return nil
	}
	// Convert internal type to common type
	return &common.RelationshipInfo{
		FieldName:    info.fieldName,
		JSONName:     info.jsonName,
		RelationType: info.relationType,
		ForeignKey:   info.foreignKey,
		References:   info.references,
		JoinTable:    info.joinTable,
		RelatedModel: info.relatedModel,
	}
}

type relationshipInfo struct {
	fieldName    string
	jsonName     string
	relationType string // "belongsTo", "hasMany", "hasOne", "many2many"
	foreignKey   string
	references   string
	joinTable    string
	relatedModel interface{}
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
