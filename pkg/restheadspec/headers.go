package restheadspec

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Warky-Devs/ResolveSpec/pkg/common"
	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
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
	AdvancedSQL    map[string]string // Column -> SQL expression
	ComputedQL     map[string]string // Column -> CQL expression
	Distinct       bool
	SkipCount      bool
	SkipCache      bool
	FetchRowNumber *string
	PKRow          *string

	// Response format
	ResponseFormat string // "simple", "detail", "syncfusion"

	// Transaction
	AtomicTransaction bool

	// Cursor pagination
	CursorForward  string
	CursorBackward string
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
	// Check for ZIP_ prefix
	if strings.HasPrefix(value, "ZIP_") {
		decoded, err := base64.StdEncoding.DecodeString(value[4:])
		if err == nil {
			return string(decoded)
		}
		logger.Warn("Failed to decode ZIP_ prefixed value: %v", err)
		return value
	}

	// Check for __ prefix
	if strings.HasPrefix(value, "__") {
		decoded, err := base64.StdEncoding.DecodeString(value[2:])
		if err == nil {
			return string(decoded)
		}
		logger.Warn("Failed to decode __ prefixed value: %v", err)
		return value
	}

	return value
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
			options.CleanJSON = strings.ToLower(decodedValue) == "true"

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
			h.parsePreload(&options, decodedValue)
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
			options.Distinct = strings.ToLower(decodedValue) == "true"
		case strings.HasPrefix(normalizedKey, "x-skipcount"):
			options.SkipCount = strings.ToLower(decodedValue) == "true"
		case strings.HasPrefix(normalizedKey, "x-skipcache"):
			options.SkipCache = strings.ToLower(decodedValue) == "true"
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
			options.AtomicTransaction = strings.ToLower(decodedValue) == "true"
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
}

// parseNotSelectFields parses x-not-select-fields header
func (h *Handler) parseNotSelectFields(options *ExtendedRequestOptions, value string) {
	if value == "" {
		return
	}
	options.OmitColumns = h.parseCommaSeparated(value)
}

// parseFieldFilter parses x-fieldfilter-{colname} header (exact match)
func (h *Handler) parseFieldFilter(options *ExtendedRequestOptions, headerKey, value string) {
	colName := strings.TrimPrefix(headerKey, "x-fieldfilter-")
	options.Filters = append(options.Filters, common.FilterOption{
		Column:   colName,
		Operator: "eq",
		Value:    value,
	})
}

// parseSearchFilter parses x-searchfilter-{colname} header (ILIKE search)
func (h *Handler) parseSearchFilter(options *ExtendedRequestOptions, headerKey, value string) {
	colName := strings.TrimPrefix(headerKey, "x-searchfilter-")
	// Use ILIKE for fuzzy search
	options.Filters = append(options.Filters, common.FilterOption{
		Column:   colName,
		Operator: "ilike",
		Value:    "%" + value + "%",
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
	filterOp := h.mapSearchOperator(operator, value)

	options.Filters = append(options.Filters, filterOp)

	// Note: OR logic would need special handling in query builder
	// For now, we'll add a comment to indicate OR logic
	if logicOp == "OR" {
		// TODO: Implement OR logic in query builder
		logger.Debug("OR logic filter: %s %s %v", colName, filterOp.Operator, filterOp.Value)
	}
}

// mapSearchOperator maps search operator names to filter operators
func (h *Handler) mapSearchOperator(operator, value string) common.FilterOption {
	operator = strings.ToLower(operator)

	switch operator {
	case "contains":
		return common.FilterOption{Operator: "ilike", Value: "%" + value + "%"}
	case "beginswith", "startswith":
		return common.FilterOption{Operator: "ilike", Value: value + "%"}
	case "endswith":
		return common.FilterOption{Operator: "ilike", Value: "%" + value}
	case "equals", "eq":
		return common.FilterOption{Operator: "eq", Value: value}
	case "notequals", "neq", "ne":
		return common.FilterOption{Operator: "neq", Value: value}
	case "greaterthan", "gt":
		return common.FilterOption{Operator: "gt", Value: value}
	case "lessthan", "lt":
		return common.FilterOption{Operator: "lt", Value: value}
	case "greaterthanorequal", "gte", "ge":
		return common.FilterOption{Operator: "gte", Value: value}
	case "lessthanorequal", "lte", "le":
		return common.FilterOption{Operator: "lte", Value: value}
	case "between":
		// Parse between values (format: "value1,value2")
		// Between is exclusive (> value1 AND < value2)
		parts := strings.Split(value, ",")
		if len(parts) == 2 {
			return common.FilterOption{Operator: "between", Value: parts}
		}
		return common.FilterOption{Operator: "eq", Value: value}
	case "betweeninclusive":
		// Parse between values (format: "value1,value2")
		// Between inclusive is >= value1 AND <= value2
		parts := strings.Split(value, ",")
		if len(parts) == 2 {
			return common.FilterOption{Operator: "between_inclusive", Value: parts}
		}
		return common.FilterOption{Operator: "eq", Value: value}
	case "in":
		// Parse IN values (format: "value1,value2,value3")
		values := strings.Split(value, ",")
		return common.FilterOption{Operator: "in", Value: values}
	case "empty", "isnull", "null":
		// Check for NULL or empty string
		return common.FilterOption{Operator: "is_null", Value: nil}
	case "notempty", "isnotnull", "notnull":
		// Check for NOT NULL
		return common.FilterOption{Operator: "is_not_null", Value: nil}
	default:
		logger.Warn("Unknown search operator: %s, defaulting to equals", operator)
		return common.FilterOption{Operator: "eq", Value: value}
	}
}

// parsePreload parses x-preload header
// Format: RelationName:field1,field2 or RelationName or multiple separated by |
func (h *Handler) parsePreload(options *ExtendedRequestOptions, value string) {
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

		if strings.HasPrefix(field, "-") {
			direction = "DESC"
			colName = strings.TrimPrefix(field, "-")
		} else if strings.HasPrefix(field, "+") {
			direction = "ASC"
			colName = strings.TrimPrefix(field, "+")
		}

		options.Sort = append(options.Sort, common.SortOption{
			Column:    colName,
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

// parseJSONHeader parses a header value as JSON
func (h *Handler) parseJSONHeader(value string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(value), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON header: %w", err)
	}
	return result, nil
}
