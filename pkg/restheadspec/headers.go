package restheadspec

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/bitechdev/ResolveSpec/pkg/common"
	"github.com/bitechdev/ResolveSpec/pkg/logger"
)

// ExtendedRequestOptions extends common.RequestOptions with additional features
type ExtendedRequestOptions struct {
	common.RequestOptions

	// Field selection
	CleanJSON bool

	// Advanced filtering
	SearchColumns  []string
	CustomSQLWhere string
	CustomSQLOr    string

	// Joins
	Expand []ExpandOption

	// Advanced features
	AdvancedSQL map[string]string // Column -> SQL expression
	ComputedQL  map[string]string // Column -> CQL expression
	Distinct    bool
	SkipCount   bool
	SkipCache   bool
	PKRow       *string

	// Response format
	ResponseFormat string // "simple", "detail", "syncfusion"

	// Single record normalization - convert single-element arrays to objects
	SingleRecordAsObject bool

	// Transaction
	AtomicTransaction bool

	// X-Files configuration - comprehensive query options as a single JSON object
	XFiles *XFiles
}

// ExpandOption represents a relation expansion configuration
type ExpandOption struct {
	Relation string
	Columns  []string
	Where    string
	Sort     string
}

// decodeHeaderValue decodes base64 encoded header values
// Supports ZIP_ and __ prefixes for base64 encoding
func decodeHeaderValue(value string) string {
	str, _ := DecodeParam(value)
	return str
}

// DecodeParam - Decodes parameter string and returns unencoded string
func DecodeParam(pStr string) (string, error) {
	var code = pStr
	if strings.HasPrefix(pStr, "ZIP_") {
		code = strings.ReplaceAll(pStr, "ZIP_", "")
		code = strings.ReplaceAll(code, "\n", "")
		code = strings.ReplaceAll(code, "\r", "")
		code = strings.ReplaceAll(code, " ", "")
		strDat, err := base64.StdEncoding.DecodeString(code)
		if err != nil {
			return code, fmt.Errorf("failed to read parameter base64: %v", err)
		} else {
			code = string(strDat)
		}
	} else if strings.HasPrefix(pStr, "__") {
		code = strings.ReplaceAll(pStr, "__", "")
		code = strings.ReplaceAll(code, "\n", "")
		code = strings.ReplaceAll(code, "\r", "")
		code = strings.ReplaceAll(code, " ", "")

		strDat, err := base64.StdEncoding.DecodeString(code)
		if err != nil {
			return code, fmt.Errorf("failed to read parameter base64: %v", err)
		} else {
			code = string(strDat)
		}
	}

	if strings.HasPrefix(code, "ZIP_") || strings.HasPrefix(code, "__") {
		code, _ = DecodeParam(code)
	}

	return code, nil
}

// parseOptionsFromHeaders parses all request options from HTTP headers
// If model is provided, it will resolve table names to field names in preload/expand options
func (h *Handler) parseOptionsFromHeaders(r common.Request, model interface{}) ExtendedRequestOptions {
	options := ExtendedRequestOptions{
		RequestOptions: common.RequestOptions{
			Filters: make([]common.FilterOption, 0),
			Sort:    make([]common.SortOption, 0),
			Preload: make([]common.PreloadOption, 0),
		},
		AdvancedSQL:          make(map[string]string),
		ComputedQL:           make(map[string]string),
		Expand:               make([]ExpandOption, 0),
		ResponseFormat:       "simple", // Default response format
		SingleRecordAsObject: true,     // Default: normalize single-element arrays to objects
	}

	// Get all headers
	headers := r.AllHeaders()

	// Get all query parameters
	queryParams := r.AllQueryParams()

	// Merge headers and query parameters - query parameters take precedence
	// This allows the same parameters to be specified in either headers or query string
	combinedParams := make(map[string]string)
	for key, value := range headers {
		combinedParams[key] = value
	}
	for key, value := range queryParams {
		combinedParams[key] = value
	}

	// Process each parameter (from both headers and query params)
	for key, value := range combinedParams {
		// Normalize parameter key to lowercase for consistent matching
		normalizedKey := strings.ToLower(key)

		// Decode value if it's base64 encoded
		decodedValue := decodeHeaderValue(value)

		// Parse based on parameter prefix/name
		switch {
		// Field Selection
		case strings.HasPrefix(normalizedKey, "x-select-fields"):
			h.parseSelectFields(&options, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-not-select-fields"):
			h.parseNotSelectFields(&options, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-clean-json"):
			options.CleanJSON = strings.EqualFold(decodedValue, "true")

		// Filtering & Search
		case strings.HasPrefix(normalizedKey, "x-fieldfilter-"):
			h.parseFieldFilter(&options, normalizedKey, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-searchfilter-"):
			h.parseSearchFilter(&options, normalizedKey, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-searchop-"):
			h.parseSearchOp(&options, normalizedKey, decodedValue, "AND")
		case strings.HasPrefix(normalizedKey, "x-searchor-"):
			h.parseSearchOp(&options, normalizedKey, decodedValue, "OR")
		case strings.HasPrefix(normalizedKey, "x-searchand-"):
			h.parseSearchOp(&options, normalizedKey, decodedValue, "AND")
		case strings.HasPrefix(normalizedKey, "x-searchcols"):
			options.SearchColumns = h.parseCommaSeparated(decodedValue)
		case strings.HasPrefix(normalizedKey, "x-custom-sql-w"):
			options.CustomSQLWhere = decodedValue
		case strings.HasPrefix(normalizedKey, "x-custom-sql-or"):
			options.CustomSQLOr = decodedValue

		// Joins & Relations
		case strings.HasPrefix(normalizedKey, "x-preload"):
			if strings.HasSuffix(normalizedKey, "-where") {
				continue
			}
			whereClaude := combinedParams[fmt.Sprintf("%s-where", key)]
			h.parsePreload(&options, decodedValue, decodeHeaderValue(whereClaude))

		case strings.HasPrefix(normalizedKey, "x-expand"):
			h.parseExpand(&options, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-custom-sql-join"):
			// TODO: Implement custom SQL join
			logger.Debug("Custom SQL join not yet implemented: %s", decodedValue)

		// Sorting & Pagination
		case strings.HasPrefix(normalizedKey, "x-sort"):
			h.parseSorting(&options, decodedValue)
		//Special cases for older clients using sort(a,b,-c) syntax
		case strings.HasPrefix(normalizedKey, "sort(") && strings.Contains(normalizedKey, ")"):
			sortValue := normalizedKey[strings.Index(normalizedKey, "(")+1 : strings.Index(normalizedKey, ")")]
			h.parseSorting(&options, sortValue)
		case strings.HasPrefix(normalizedKey, "x-limit"):
			if limit, err := strconv.Atoi(decodedValue); err == nil {
				options.Limit = &limit
			}
		//Special cases for older clients using limit(n) syntax
		case strings.HasPrefix(normalizedKey, "limit(") && strings.Contains(normalizedKey, ")"):
			limitValue := normalizedKey[strings.Index(normalizedKey, "(")+1 : strings.Index(normalizedKey, ")")]
			limitValueParts := strings.Split(limitValue, ",")

			if len(limitValueParts) > 1 {
				if offset, err := strconv.Atoi(limitValueParts[0]); err == nil {
					options.Offset = &offset
				}
				if limit, err := strconv.Atoi(limitValueParts[1]); err == nil {
					options.Limit = &limit
				}
			} else {
				if limit, err := strconv.Atoi(limitValueParts[0]); err == nil {
					options.Limit = &limit
				}
			}

		case strings.HasPrefix(normalizedKey, "x-offset"):
			if offset, err := strconv.Atoi(decodedValue); err == nil {
				options.Offset = &offset
			}

		case strings.HasPrefix(normalizedKey, "x-cursor-forward"):
			options.CursorForward = decodedValue
		case strings.HasPrefix(normalizedKey, "x-cursor-backward"):
			options.CursorBackward = decodedValue

		// Advanced Features
		case strings.HasPrefix(normalizedKey, "x-advsql-"):
			colName := strings.TrimPrefix(normalizedKey, "x-advsql-")
			options.AdvancedSQL[colName] = decodedValue
		case strings.HasPrefix(normalizedKey, "x-cql-sel-"):
			colName := strings.TrimPrefix(normalizedKey, "x-cql-sel-")
			options.ComputedQL[colName] = decodedValue
		case strings.HasPrefix(normalizedKey, "x-distinct"):
			options.Distinct = strings.EqualFold(decodedValue, "true")
		case strings.HasPrefix(normalizedKey, "x-skipcount"):
			options.SkipCount = strings.EqualFold(decodedValue, "true")
		case strings.HasPrefix(normalizedKey, "x-skipcache"):
			options.SkipCache = strings.EqualFold(decodedValue, "true")
		case strings.HasPrefix(normalizedKey, "x-fetch-rownumber"):
			options.FetchRowNumber = &decodedValue
		case strings.HasPrefix(normalizedKey, "x-pkrow"):
			options.PKRow = &decodedValue

		// Response Format
		case strings.HasPrefix(normalizedKey, "x-simpleapi"):
			options.ResponseFormat = "simple"
		case strings.HasPrefix(normalizedKey, "x-detailapi"):
			options.ResponseFormat = "detail"
		case strings.HasPrefix(normalizedKey, "x-syncfusion"):
			options.ResponseFormat = "syncfusion"
		case strings.HasPrefix(normalizedKey, "x-single-record-as-object"):
			// Parse as boolean - "false" disables, "true" enables (default is true)
			if strings.EqualFold(decodedValue, "false") {
				options.SingleRecordAsObject = false
			} else if strings.EqualFold(decodedValue, "true") {
				options.SingleRecordAsObject = true
			}

		// Transaction Control
		case strings.HasPrefix(normalizedKey, "x-transaction-atomic"):
			options.AtomicTransaction = strings.EqualFold(decodedValue, "true")

		// X-Files - comprehensive JSON configuration
		case strings.HasPrefix(normalizedKey, "x-files"):
			h.parseXFiles(&options, decodedValue)
		}
	}

	// Resolve relation names (convert table names to field names) if model is provided
	if model != nil {
		h.resolveRelationNamesInOptions(&options, model)
	}

	return options
}

// parseSelectFields parses x-select-fields header
func (h *Handler) parseSelectFields(options *ExtendedRequestOptions, value string) {
	if value == "" {
		return
	}
	options.Columns = h.parseCommaSeparated(value)
	if len(options.Columns) > 1 {
		options.CleanJSON = true
	}
}

// parseNotSelectFields parses x-not-select-fields header
func (h *Handler) parseNotSelectFields(options *ExtendedRequestOptions, value string) {
	if value == "" {
		return
	}
	options.OmitColumns = h.parseCommaSeparated(value)
	if len(options.OmitColumns) > 1 {
		options.CleanJSON = true
	}
}

// parseFieldFilter parses x-fieldfilter-{colname} header (exact match)
func (h *Handler) parseFieldFilter(options *ExtendedRequestOptions, headerKey, value string) {
	colName := strings.TrimPrefix(headerKey, "x-fieldfilter-")
	options.Filters = append(options.Filters, common.FilterOption{
		Column:        colName,
		Operator:      "eq",
		Value:         value,
		LogicOperator: "AND", // Default to AND
	})
}

// parseSearchFilter parses x-searchfilter-{colname} header (ILIKE search)
func (h *Handler) parseSearchFilter(options *ExtendedRequestOptions, headerKey, value string) {
	colName := strings.TrimPrefix(headerKey, "x-searchfilter-")
	// Use ILIKE for fuzzy search
	options.Filters = append(options.Filters, common.FilterOption{
		Column:        colName,
		Operator:      "ilike",
		Value:         "%" + value + "%",
		LogicOperator: "AND", // Default to AND
	})
}

// parseSearchOp parses x-searchop-{operator}-{colname} and x-searchor-{operator}-{colname}
func (h *Handler) parseSearchOp(options *ExtendedRequestOptions, headerKey, value, logicOp string) {
	// Extract operator and column name
	// Format: x-searchop-{operator}-{colname} or x-searchor-{operator}-{colname}
	var prefix string
	if logicOp == "OR" {
		prefix = "x-searchor-"
	} else {
		prefix = "x-searchop-"
		if strings.HasPrefix(headerKey, "x-searchand-") {
			prefix = "x-searchand-"
		}
	}

	rest := strings.TrimPrefix(headerKey, prefix)
	parts := strings.SplitN(rest, "-", 2)
	if len(parts) != 2 {
		logger.Warn("Invalid search operator header format: %s", headerKey)
		return
	}

	operator := parts[0]
	colName := parts[1]

	// Map operator names to filter operators
	filterOp := h.mapSearchOperator(colName, operator, value)

	// Set the logic operator (AND or OR)
	filterOp.LogicOperator = logicOp

	options.Filters = append(options.Filters, filterOp)

	logger.Debug("%s logic filter: %s %s %v", logicOp, colName, filterOp.Operator, filterOp.Value)
}

// mapSearchOperator maps search operator names to filter operators
func (h *Handler) mapSearchOperator(colName, operator, value string) common.FilterOption {
	operator = strings.ToLower(operator)

	switch operator {
	case "contains", "contain", "like":
		return common.FilterOption{Column: colName, Operator: "ilike", Value: "%" + value + "%"}
	case "beginswith", "startswith":
		return common.FilterOption{Column: colName, Operator: "ilike", Value: value + "%"}
	case "endswith":
		return common.FilterOption{Column: colName, Operator: "ilike", Value: "%" + value}
	case "equals", "eq", "=":
		return common.FilterOption{Column: colName, Operator: "eq", Value: value}
	case "notequals", "neq", "ne", "!=", "<>":
		return common.FilterOption{Column: colName, Operator: "neq", Value: value}
	case "greaterthan", "gt", ">":
		return common.FilterOption{Column: colName, Operator: "gt", Value: value}
	case "lessthan", "lt", "<":
		return common.FilterOption{Column: colName, Operator: "lt", Value: value}
	case "greaterthanorequal", "gte", "ge", ">=":
		return common.FilterOption{Column: colName, Operator: "gte", Value: value}
	case "lessthanorequal", "lte", "le", "<=":
		return common.FilterOption{Column: colName, Operator: "lte", Value: value}
	case "between":
		// Parse between values (format: "value1,value2")
		// Between is exclusive (> value1 AND < value2)
		parts := strings.Split(value, ",")
		if len(parts) == 2 {
			return common.FilterOption{Column: colName, Operator: "between", Value: parts}
		}
		return common.FilterOption{Column: colName, Operator: "eq", Value: value}
	case "betweeninclusive":
		// Parse between values (format: "value1,value2")
		// Between inclusive is >= value1 AND <= value2
		parts := strings.Split(value, ",")
		if len(parts) == 2 {
			return common.FilterOption{Column: colName, Operator: "between_inclusive", Value: parts}
		}
		return common.FilterOption{Column: colName, Operator: "eq", Value: value}
	case "in":
		// Parse IN values (format: "value1,value2,value3")
		values := strings.Split(value, ",")
		return common.FilterOption{Column: colName, Operator: "in", Value: values}
	case "empty", "isnull", "null":
		// Check for NULL or empty string
		return common.FilterOption{Column: colName, Operator: "is_null", Value: nil}
	case "notempty", "isnotnull", "notnull":
		// Check for NOT NULL
		return common.FilterOption{Column: colName, Operator: "is_not_null", Value: nil}
	default:
		logger.Warn("Unknown search operator: %s, defaulting to equals", operator)
		return common.FilterOption{Column: colName, Operator: "eq", Value: value}
	}
}

// parsePreload parses x-preload header
// Format: RelationName:field1,field2 or RelationName or multiple separated by |
func (h *Handler) parsePreload(options *ExtendedRequestOptions, values ...string) {
	if len(values) == 0 {
		return
	}
	value := values[0]
	whereClause := ""
	if len(values) > 1 {
		whereClause = values[1]
	}
	if value == "" {
		return
	}

	// Split by | for multiple preloads
	preloads := strings.Split(value, "|")
	for _, preloadStr := range preloads {
		preloadStr = strings.TrimSpace(preloadStr)
		if preloadStr == "" {
			continue
		}

		// Parse relation:columns format
		parts := strings.SplitN(preloadStr, ":", 2)
		preload := common.PreloadOption{
			Relation: strings.TrimSpace(parts[0]),
			Where:    whereClause,
		}

		if len(parts) == 2 {
			// Parse columns
			preload.Columns = h.parseCommaSeparated(parts[1])
		}

		options.Preload = append(options.Preload, preload)
	}
}

// parseExpand parses x-expand header (LEFT JOIN expansion)
// Format: RelationName:field1,field2 or RelationName or multiple separated by |
func (h *Handler) parseExpand(options *ExtendedRequestOptions, value string) {
	if value == "" {
		return
	}

	// Split by | for multiple expands
	expands := strings.Split(value, "|")
	for _, expandStr := range expands {
		expandStr = strings.TrimSpace(expandStr)
		if expandStr == "" {
			continue
		}

		// Parse relation:columns format
		parts := strings.SplitN(expandStr, ":", 2)
		expand := ExpandOption{
			Relation: strings.TrimSpace(parts[0]),
		}

		if len(parts) == 2 {
			// Parse columns
			expand.Columns = h.parseCommaSeparated(parts[1])
		}

		options.Expand = append(options.Expand, expand)
	}
}

// parseSorting parses x-sort header
// Format: +field1,-field2,field3 (+ for ASC, - for DESC, default ASC)
func (h *Handler) parseSorting(options *ExtendedRequestOptions, value string) {
	if value == "" {
		return
	}

	sortFields := h.parseCommaSeparated(value)
	for _, field := range sortFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		direction := "ASC"
		colName := field

		switch {
		case strings.HasPrefix(field, "-"):
			direction = "DESC"
			colName = strings.TrimPrefix(field, "-")
		case strings.HasPrefix(field, "+"):
			direction = "ASC"
			colName = strings.TrimPrefix(field, "+")
		case strings.HasSuffix(field, " desc"):
			direction = "DESC"
			colName = strings.TrimSuffix(field, "desc")
		case strings.HasSuffix(field, " asc"):
			direction = "ASC"
			colName = strings.TrimSuffix(field, "asc")
		}

		options.Sort = append(options.Sort, common.SortOption{
			Column:    strings.Trim(colName, " "),
			Direction: direction,
		})
	}
}

// parseCommaSeparated parses comma-separated values and trims whitespace
func (h *Handler) parseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// parseXFiles parses x-files header containing comprehensive JSON configuration
// and populates ExtendedRequestOptions fields from it
func (h *Handler) parseXFiles(options *ExtendedRequestOptions, value string) {
	if value == "" {
		return
	}

	var xfiles XFiles
	if err := json.Unmarshal([]byte(value), &xfiles); err != nil {
		logger.Warn("Failed to parse x-files header: %v", err)
		return
	}

	logger.Debug("Parsed x-files configuration for table: %s", xfiles.TableName)

	// Store the original XFiles for reference
	options.XFiles = &xfiles

	// Map XFiles fields to ExtendedRequestOptions

	// Column selection
	if len(xfiles.Columns) > 0 {
		options.Columns = append(options.Columns, xfiles.Columns...)
		logger.Debug("X-Files: Added columns: %v", xfiles.Columns)
	}

	// Omit columns
	if len(xfiles.OmitColumns) > 0 {
		options.OmitColumns = append(options.OmitColumns, xfiles.OmitColumns...)
		logger.Debug("X-Files: Added omit columns: %v", xfiles.OmitColumns)
	}

	// Computed columns (CQL) -> ComputedQL
	if len(xfiles.CQLColumns) > 0 {
		if options.ComputedQL == nil {
			options.ComputedQL = make(map[string]string)
		}
		for i, cqlExpr := range xfiles.CQLColumns {
			colName := fmt.Sprintf("cql%d", i+1)
			options.ComputedQL[colName] = cqlExpr
			logger.Debug("X-Files: Added computed column %s: %s", colName, cqlExpr)
		}
	}

	// Sorting
	if len(xfiles.Sort) > 0 {
		for _, sortField := range xfiles.Sort {
			direction := "ASC"
			colName := sortField

			// Handle direction prefixes
			if strings.HasPrefix(sortField, "-") {
				direction = "DESC"
				colName = strings.TrimPrefix(sortField, "-")
			} else if strings.HasPrefix(sortField, "+") {
				colName = strings.TrimPrefix(sortField, "+")
			}

			// Handle DESC suffix
			if strings.HasSuffix(strings.ToLower(colName), " desc") {
				direction = "DESC"
				colName = strings.TrimSuffix(strings.ToLower(colName), " desc")
			} else if strings.HasSuffix(strings.ToLower(colName), " asc") {
				colName = strings.TrimSuffix(strings.ToLower(colName), " asc")
			}

			options.Sort = append(options.Sort, common.SortOption{
				Column:    strings.TrimSpace(colName),
				Direction: direction,
			})
		}
		logger.Debug("X-Files: Added %d sort options", len(xfiles.Sort))
	}

	// Filter fields
	if len(xfiles.FilterFields) > 0 {
		for _, filterField := range xfiles.FilterFields {
			options.Filters = append(options.Filters, common.FilterOption{
				Column:        filterField.Field,
				Operator:      filterField.Operator,
				Value:         filterField.Value,
				LogicOperator: "AND", // Default to AND
			})
		}
		logger.Debug("X-Files: Added %d filter fields", len(xfiles.FilterFields))
	}

	// SQL AND conditions -> CustomSQLWhere
	if len(xfiles.SqlAnd) > 0 {
		if options.CustomSQLWhere != "" {
			options.CustomSQLWhere += " AND "
		}
		options.CustomSQLWhere += "(" + strings.Join(xfiles.SqlAnd, " AND ") + ")"
		logger.Debug("X-Files: Added SQL AND conditions")
	}

	// SQL OR conditions -> CustomSQLOr
	if len(xfiles.SqlOr) > 0 {
		if options.CustomSQLOr != "" {
			options.CustomSQLOr += " OR "
		}
		options.CustomSQLOr += "(" + strings.Join(xfiles.SqlOr, " OR ") + ")"
		logger.Debug("X-Files: Added SQL OR conditions")
	}

	// Pagination - Limit
	if limitStr := xfiles.Limit.String(); limitStr != "" && limitStr != "0" {
		if limitVal, err := xfiles.Limit.Int64(); err == nil && limitVal > 0 {
			limit := int(limitVal)
			options.Limit = &limit
			logger.Debug("X-Files: Set limit: %d", limit)
		}
	}

	// Pagination - Offset
	if offsetStr := xfiles.Offset.String(); offsetStr != "" && offsetStr != "0" {
		if offsetVal, err := xfiles.Offset.Int64(); err == nil && offsetVal > 0 {
			offset := int(offsetVal)
			options.Offset = &offset
			logger.Debug("X-Files: Set offset: %d", offset)
		}
	}

	// Cursor pagination
	if xfiles.CursorForward != "" {
		options.CursorForward = xfiles.CursorForward
		logger.Debug("X-Files: Set cursor forward")
	}
	if xfiles.CursorBackward != "" {
		options.CursorBackward = xfiles.CursorBackward
		logger.Debug("X-Files: Set cursor backward")
	}

	// Flags
	if xfiles.Skipcount {
		options.SkipCount = true
		logger.Debug("X-Files: Set skip count")
	}

	// Process ParentTables and ChildTables recursively
	h.processXFilesRelations(&xfiles, options, "")
}

// processXFilesRelations processes ParentTables and ChildTables from XFiles
// and adds them as Preload options recursively
func (h *Handler) processXFilesRelations(xfiles *XFiles, options *ExtendedRequestOptions, basePath string) {
	if xfiles == nil {
		return
	}

	// Process ParentTables
	if len(xfiles.ParentTables) > 0 {
		logger.Debug("X-Files: Processing %d parent tables", len(xfiles.ParentTables))
		for _, parentTable := range xfiles.ParentTables {
			h.addXFilesPreload(parentTable, options, basePath)
		}
	}

	// Process ChildTables
	if len(xfiles.ChildTables) > 0 {
		logger.Debug("X-Files: Processing %d child tables", len(xfiles.ChildTables))
		for _, childTable := range xfiles.ChildTables {
			h.addXFilesPreload(childTable, options, basePath)
		}
	}
}

// resolveRelationNamesInOptions resolves all table names to field names in preload options
// This is called internally by parseOptionsFromHeaders when a model is provided
func (h *Handler) resolveRelationNamesInOptions(options *ExtendedRequestOptions, model interface{}) {
	if options == nil || model == nil {
		return
	}

	// Resolve relation names in all preload options
	for i := range options.Preload {
		preload := &options.Preload[i]

		// Split the relation path (e.g., "parent.child.grandchild")
		parts := strings.Split(preload.Relation, ".")
		resolvedParts := make([]string, 0, len(parts))

		// Resolve each part of the path
		currentModel := model
		for _, part := range parts {
			resolvedPart := h.resolveRelationName(currentModel, part)
			resolvedParts = append(resolvedParts, resolvedPart)

			// Try to get the model type for the next level
			// This allows nested resolution
			if nextModel := h.getRelationModel(currentModel, resolvedPart); nextModel != nil {
				currentModel = nextModel
			}
		}

		// Update the relation path with resolved names
		resolvedPath := strings.Join(resolvedParts, ".")
		if resolvedPath != preload.Relation {
			logger.Debug("Resolved relation path '%s' -> '%s'", preload.Relation, resolvedPath)
			preload.Relation = resolvedPath
		}
	}

	// Resolve relation names in expand options
	for i := range options.Expand {
		expand := &options.Expand[i]
		resolved := h.resolveRelationName(model, expand.Relation)
		if resolved != expand.Relation {
			logger.Debug("Resolved expand relation '%s' -> '%s'", expand.Relation, resolved)
			expand.Relation = resolved
		}
	}
}

// getRelationModel gets the model type for a relation field
func (h *Handler) getRelationModel(model interface{}, fieldName string) interface{} {
	if model == nil || fieldName == "" {
		return nil
	}

	modelType := reflect.TypeOf(model)
	if modelType == nil {
		return nil
	}

	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType == nil || modelType.Kind() != reflect.Struct {
		return nil
	}

	// Find the field
	field, found := modelType.FieldByName(fieldName)
	if !found {
		return nil
	}

	// Get the target type
	targetType := field.Type
	if targetType == nil {
		return nil
	}

	if targetType.Kind() == reflect.Slice {
		targetType = targetType.Elem()
		if targetType == nil {
			return nil
		}
	}
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
		if targetType == nil {
			return nil
		}
	}

	if targetType.Kind() != reflect.Struct {
		return nil
	}

	// Create a zero value of the target type
	return reflect.New(targetType).Elem().Interface()
}

// resolveRelationName resolves a relation name or table name to the actual field name in the model
// If the input is already a field name, it returns it as-is
// If the input is a table name, it looks up the corresponding relation field
func (h *Handler) resolveRelationName(model interface{}, nameOrTable string) string {
	if model == nil || nameOrTable == "" {
		return nameOrTable
	}

	modelType := reflect.TypeOf(model)
	if modelType == nil {
		return nameOrTable
	}

	// Dereference pointer if needed
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// Check again after dereferencing
	if modelType == nil {
		return nameOrTable
	}

	// Ensure it's a struct
	if modelType.Kind() != reflect.Struct {
		return nameOrTable
	}

	// First, check if the input matches a field name directly
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if field.Name == nameOrTable {
			// It's already a field name
			logger.Debug("Input '%s' is a field name", nameOrTable)
			return nameOrTable
		}
	}

	// If not found as a field name, try to look it up as a table name
	normalizedInput := strings.ToLower(strings.ReplaceAll(nameOrTable, "_", ""))

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		fieldType := field.Type

		// Check if it's a slice or pointer to a struct
		var targetType reflect.Type
		if fieldType.Kind() == reflect.Slice {
			targetType = fieldType.Elem()
		} else if fieldType.Kind() == reflect.Ptr {
			targetType = fieldType.Elem()
		}

		if targetType != nil {
			// Dereference pointer if the slice contains pointers
			if targetType.Kind() == reflect.Ptr {
				targetType = targetType.Elem()
			}

			// Check if it's a struct type
			if targetType.Kind() == reflect.Struct {
				// Get the type name and normalize it
				typeName := targetType.Name()

				// Extract the table name from type name
				// Patterns: ModelCoreMastertaskitem -> mastertaskitem
				//           ModelMastertaskitem -> mastertaskitem
				normalizedTypeName := strings.ToLower(typeName)

				// Remove common prefixes like "model", "modelcore", etc.
				normalizedTypeName = strings.TrimPrefix(normalizedTypeName, "modelcore")
				normalizedTypeName = strings.TrimPrefix(normalizedTypeName, "model")

				// Compare normalized names
				if normalizedTypeName == normalizedInput {
					logger.Debug("Resolved table name '%s' to field '%s' (type: %s)", nameOrTable, field.Name, typeName)
					return field.Name
				}
			}
		}
	}

	// If no match found, return the original input
	logger.Debug("No field found for '%s', using as-is", nameOrTable)
	return nameOrTable
}

// addXFilesPreload converts an XFiles relation into a PreloadOption
// and recursively processes its children
func (h *Handler) addXFilesPreload(xfile *XFiles, options *ExtendedRequestOptions, basePath string) {
	if xfile == nil || xfile.TableName == "" {
		return
	}

	// Store the table name as-is for now - it will be resolved to field name later
	// when we have the model instance available
	relationPath := xfile.TableName
	if basePath != "" {
		relationPath = basePath + "." + xfile.TableName
	}

	logger.Debug("X-Files: Adding preload for relation: %s", relationPath)

	// Create PreloadOption from XFiles configuration
	preloadOpt := common.PreloadOption{
		Relation:    relationPath,
		Columns:     xfile.Columns,
		OmitColumns: xfile.OmitColumns,
	}

	// Add sorting if specified
	if len(xfile.Sort) > 0 {
		preloadOpt.Sort = make([]common.SortOption, 0, len(xfile.Sort))
		for _, sortField := range xfile.Sort {
			direction := "ASC"
			colName := sortField

			// Handle direction prefixes
			if strings.HasPrefix(sortField, "-") {
				direction = "DESC"
				colName = strings.TrimPrefix(sortField, "-")
			} else if strings.HasPrefix(sortField, "+") {
				colName = strings.TrimPrefix(sortField, "+")
			}

			preloadOpt.Sort = append(preloadOpt.Sort, common.SortOption{
				Column:    strings.TrimSpace(colName),
				Direction: direction,
			})
		}
	}

	// Add filters if specified
	if len(xfile.FilterFields) > 0 {
		preloadOpt.Filters = make([]common.FilterOption, 0, len(xfile.FilterFields))
		for _, filterField := range xfile.FilterFields {
			preloadOpt.Filters = append(preloadOpt.Filters, common.FilterOption{
				Column:        filterField.Field,
				Operator:      filterField.Operator,
				Value:         filterField.Value,
				LogicOperator: "AND",
			})
		}
	}

	// Add WHERE clause if SQL conditions specified
	whereConditions := make([]string, 0)
	if len(xfile.SqlAnd) > 0 {
		whereConditions = append(whereConditions, xfile.SqlAnd...)
	}
	if len(whereConditions) > 0 {
		preloadOpt.Where = strings.Join(whereConditions, " AND ")
	}

	// Add limit if specified
	if limitStr := xfile.Limit.String(); limitStr != "" && limitStr != "0" {
		if limitVal, err := xfile.Limit.Int64(); err == nil && limitVal > 0 {
			limit := int(limitVal)
			preloadOpt.Limit = &limit
		}
	}

	// Add computed columns (CQL) -> ComputedQL
	if len(xfile.CQLColumns) > 0 {
		preloadOpt.ComputedQL = make(map[string]string)
		for i, cqlExpr := range xfile.CQLColumns {
			colName := fmt.Sprintf("cql%d", i+1)
			preloadOpt.ComputedQL[colName] = cqlExpr
			logger.Debug("X-Files: Added computed column %s to preload %s: %s", colName, relationPath, cqlExpr)
		}
	}

	// Set recursive flag
	preloadOpt.Recursive = xfile.Recursive

	// Add the preload option
	options.Preload = append(options.Preload, preloadOpt)

	// Recursively process nested ParentTables and ChildTables
	if xfile.Recursive {
		logger.Debug("X-Files: Recursive preload enabled for: %s", relationPath)
		h.processXFilesRelations(xfile, options, relationPath)
	} else if len(xfile.ParentTables) > 0 || len(xfile.ChildTables) > 0 {
		h.processXFilesRelations(xfile, options, relationPath)
	}
}

// extractSourceColumn extracts the base column name from PostgreSQL JSON operators
// Examples:
//   - "columna->>'val'" returns "columna"
//   - "columna->'key'" returns "columna"
//   - "columna" returns "columna"
//   - "table.columna->>'val'" returns "table.columna"
func extractSourceColumn(colName string) string {
	// Check for PostgreSQL JSON operators: -> and ->>
	if idx := strings.Index(colName, "->>"); idx != -1 {
		return strings.TrimSpace(colName[:idx])
	}
	if idx := strings.Index(colName, "->"); idx != -1 {
		return strings.TrimSpace(colName[:idx])
	}
	return colName
}

// getColumnTypeFromModel uses reflection to determine the Go type of a column in a model
func (h *Handler) getColumnTypeFromModel(model interface{}, colName string) reflect.Kind {
	if model == nil {
		return reflect.Invalid
	}

	// Extract the source column name (remove JSON operators like ->> or ->)
	sourceColName := extractSourceColumn(colName)

	modelType := reflect.TypeOf(model)
	// Dereference pointer if needed
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// Ensure it's a struct
	if modelType.Kind() != reflect.Struct {
		return reflect.Invalid
	}

	// Find the field by JSON tag or field name
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Check JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			// Parse JSON tag (format: "name,omitempty")
			parts := strings.Split(jsonTag, ",")
			if parts[0] == sourceColName {
				return field.Type.Kind()
			}
		}

		// Check field name (case-insensitive)
		if strings.EqualFold(field.Name, sourceColName) {
			return field.Type.Kind()
		}

		// Check snake_case conversion
		snakeCaseName := toSnakeCase(field.Name)
		if snakeCaseName == sourceColName {
			return field.Type.Kind()
		}
	}

	return reflect.Invalid
}

// toSnakeCase converts a string from CamelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isNumericType checks if a reflect.Kind is a numeric type
func isNumericType(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 ||
		kind == reflect.Int32 || kind == reflect.Int64 || kind == reflect.Uint ||
		kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 ||
		kind == reflect.Uint64 || kind == reflect.Float32 || kind == reflect.Float64
}

// isStringType checks if a reflect.Kind is a string type
func isStringType(kind reflect.Kind) bool {
	return kind == reflect.String
}

// convertToNumericType converts a string value to the appropriate numeric type
func convertToNumericType(value string, kind reflect.Kind) (interface{}, error) {
	value = strings.TrimSpace(value)

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Parse as integer
		bitSize := 64
		switch kind {
		case reflect.Int8:
			bitSize = 8
		case reflect.Int16:
			bitSize = 16
		case reflect.Int32:
			bitSize = 32
		}

		intVal, err := strconv.ParseInt(value, 10, bitSize)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value: %w", err)
		}

		// Return the appropriate type
		switch kind {
		case reflect.Int:
			return int(intVal), nil
		case reflect.Int8:
			return int8(intVal), nil
		case reflect.Int16:
			return int16(intVal), nil
		case reflect.Int32:
			return int32(intVal), nil
		case reflect.Int64:
			return intVal, nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Parse as unsigned integer
		bitSize := 64
		switch kind {
		case reflect.Uint8:
			bitSize = 8
		case reflect.Uint16:
			bitSize = 16
		case reflect.Uint32:
			bitSize = 32
		}

		uintVal, err := strconv.ParseUint(value, 10, bitSize)
		if err != nil {
			return nil, fmt.Errorf("invalid unsigned integer value: %w", err)
		}

		// Return the appropriate type
		switch kind {
		case reflect.Uint:
			return uint(uintVal), nil
		case reflect.Uint8:
			return uint8(uintVal), nil
		case reflect.Uint16:
			return uint16(uintVal), nil
		case reflect.Uint32:
			return uint32(uintVal), nil
		case reflect.Uint64:
			return uintVal, nil
		}

	case reflect.Float32, reflect.Float64:
		// Parse as float
		bitSize := 64
		if kind == reflect.Float32 {
			bitSize = 32
		}

		floatVal, err := strconv.ParseFloat(value, bitSize)
		if err != nil {
			return nil, fmt.Errorf("invalid float value: %w", err)
		}

		if kind == reflect.Float32 {
			return float32(floatVal), nil
		}
		return floatVal, nil
	}

	return nil, fmt.Errorf("unsupported numeric type: %v", kind)
}

// isNumericValue checks if a string value can be parsed as a number
func isNumericValue(value string) bool {
	value = strings.TrimSpace(value)
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

// ColumnCastInfo holds information about whether a column needs casting
type ColumnCastInfo struct {
	NeedsCast     bool
	IsNumericType bool
}

// ValidateAndAdjustFilterForColumnType validates and adjusts a filter based on column type
// Returns ColumnCastInfo indicating whether the column should be cast to text in SQL
func (h *Handler) ValidateAndAdjustFilterForColumnType(filter *common.FilterOption, model interface{}) ColumnCastInfo {
	if filter == nil || model == nil {
		return ColumnCastInfo{NeedsCast: false, IsNumericType: false}
	}

	colType := h.getColumnTypeFromModel(model, filter.Column)
	if colType == reflect.Invalid {
		// Column not found in model, no casting needed
		logger.Debug("Column %s not found in model, skipping type validation", filter.Column)
		return ColumnCastInfo{NeedsCast: false, IsNumericType: false}
	}

	// Check if the input value is numeric
	valueIsNumeric := false
	if strVal, ok := filter.Value.(string); ok {
		strVal = strings.Trim(strVal, "%")
		valueIsNumeric = isNumericValue(strVal)
	}

	// Adjust based on column type
	switch {
	case isNumericType(colType):
		// Column is numeric
		if valueIsNumeric {
			// Value is numeric - try to convert it
			if strVal, ok := filter.Value.(string); ok {
				strVal = strings.Trim(strVal, "%")
				numericVal, err := convertToNumericType(strVal, colType)
				if err != nil {
					logger.Debug("Failed to convert value '%s' to numeric type for column %s, will use text cast", strVal, filter.Column)
					return ColumnCastInfo{NeedsCast: true, IsNumericType: true}
				}
				filter.Value = numericVal
			}
			// No cast needed - numeric column with numeric value
			return ColumnCastInfo{NeedsCast: false, IsNumericType: true}
		} else {
			// Value is not numeric - cast column to text for comparison
			logger.Debug("Non-numeric value for numeric column %s, will cast to text", filter.Column)
			return ColumnCastInfo{NeedsCast: true, IsNumericType: true}
		}

	case isStringType(colType):
		// String columns don't need casting
		return ColumnCastInfo{NeedsCast: false, IsNumericType: false}

	default:
		// For bool, time.Time, and other complex types - cast to text
		logger.Debug("Complex type column %s, will cast to text", filter.Column)
		return ColumnCastInfo{NeedsCast: true, IsNumericType: false}
	}
}
