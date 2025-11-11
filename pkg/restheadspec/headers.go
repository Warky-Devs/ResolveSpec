package restheadspec

import (
	"encoding/base64"
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

	// Transaction
	AtomicTransaction bool
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
func (h *Handler) parseOptionsFromHeaders(r common.Request) ExtendedRequestOptions {
	options := ExtendedRequestOptions{
		RequestOptions: common.RequestOptions{
			Filters: make([]common.FilterOption, 0),
			Sort:    make([]common.SortOption, 0),
			Preload: make([]common.PreloadOption, 0),
		},
		AdvancedSQL:    make(map[string]string),
		ComputedQL:     make(map[string]string),
		Expand:         make([]ExpandOption, 0),
		ResponseFormat: "simple", // Default response format
	}

	// Get all headers
	headers := r.AllHeaders()

	// Process each header
	for key, value := range headers {
		// Normalize header key to lowercase for consistent matching
		normalizedKey := strings.ToLower(key)

		// Decode value if it's base64 encoded
		decodedValue := decodeHeaderValue(value)

		// Parse based on header prefix/name
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
			whereClaude := headers[fmt.Sprintf("%s-where", key)]
			h.parsePreload(&options, decodedValue, decodeHeaderValue(whereClaude))

		case strings.HasPrefix(normalizedKey, "x-expand"):
			h.parseExpand(&options, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-custom-sql-join"):
			// TODO: Implement custom SQL join
			logger.Debug("Custom SQL join not yet implemented: %s", decodedValue)

		// Sorting & Pagination
		case strings.HasPrefix(normalizedKey, "x-sort"):
			h.parseSorting(&options, decodedValue)
		case strings.HasPrefix(normalizedKey, "x-limit"):
			if limit, err := strconv.Atoi(decodedValue); err == nil {
				options.Limit = &limit
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

		// Transaction Control
		case strings.HasPrefix(normalizedKey, "x-transaction-atomic"):
			options.AtomicTransaction = strings.EqualFold(decodedValue, "true")
		}
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

// getColumnTypeFromModel uses reflection to determine the Go type of a column in a model
func (h *Handler) getColumnTypeFromModel(model interface{}, colName string) reflect.Kind {
	if model == nil {
		return reflect.Invalid
	}

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
			if parts[0] == colName {
				return field.Type.Kind()
			}
		}

		// Check field name (case-insensitive)
		if strings.EqualFold(field.Name, colName) {
			return field.Type.Kind()
		}

		// Check snake_case conversion
		snakeCaseName := toSnakeCase(field.Name)
		if snakeCaseName == colName {
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
