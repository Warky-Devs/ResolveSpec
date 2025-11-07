package common

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
)

// ColumnValidator validates column names against a model's fields
type ColumnValidator struct {
	validColumns map[string]bool
	model        interface{}
}

// NewColumnValidator creates a new column validator for a given model
func NewColumnValidator(model interface{}) *ColumnValidator {
	validator := &ColumnValidator{
		validColumns: make(map[string]bool),
		model:        model,
	}
	validator.buildValidColumns()
	return validator
}

// buildValidColumns extracts all valid column names from the model using reflection
func (v *ColumnValidator) buildValidColumns() {
	modelType := reflect.TypeOf(v.model)

	// Unwrap pointers, slices, and arrays to get to the base struct type
	for modelType != nil && (modelType.Kind() == reflect.Ptr || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) {
		modelType = modelType.Elem()
	}

	// Validate that we have a struct type
	if modelType == nil || modelType.Kind() != reflect.Struct {
		return
	}

	// Extract column names from struct fields
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		if !field.IsExported() {
			continue
		}

		// Get column name from bun, gorm, or json tag
		columnName := v.getColumnName(field)
		if columnName != "" && columnName != "-" {
			v.validColumns[strings.ToLower(columnName)] = true
		}
	}
}

// getColumnName extracts the column name from a struct field's tags
// Supports both Bun and GORM tags
func (v *ColumnValidator) getColumnName(field reflect.StructField) string {
	// First check Bun tag for column name
	bunTag := field.Tag.Get("bun")
	if bunTag != "" && bunTag != "-" {
		parts := strings.Split(bunTag, ",")
		// The first part is usually the column name
		columnName := strings.TrimSpace(parts[0])
		if columnName != "" && columnName != "-" {
			return columnName
		}
	}

	// Check GORM tag for column name
	gormTag := field.Tag.Get("gorm")
	if strings.Contains(gormTag, "column:") {
		parts := strings.Split(gormTag, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "column:") {
				return strings.TrimPrefix(part, "column:")
			}
		}
	}

	// Fall back to JSON tag
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		// Extract just the name part (before any comma)
		jsonName := strings.Split(jsonTag, ",")[0]
		return jsonName
	}

	// Fall back to field name in lowercase (snake_case conversion would be better)
	return strings.ToLower(field.Name)
}

// ValidateColumn validates a single column name
// Returns nil if valid, error if invalid
// Columns prefixed with "cql" (case insensitive) are always valid
func (v *ColumnValidator) ValidateColumn(column string) error {
	// Allow empty columns
	if column == "" {
		return nil
	}

	// Allow columns prefixed with "cql" (case insensitive) for computed columns
	if strings.HasPrefix(strings.ToLower(column), "cql") {
		return nil
	}

	// Check if column exists in model
	if _, exists := v.validColumns[strings.ToLower(column)]; !exists {
		return fmt.Errorf("invalid column '%s': column does not exist in model", column)
	}

	return nil
}

// IsValidColumn checks if a column is valid
// Returns true if valid, false if invalid
func (v *ColumnValidator) IsValidColumn(column string) bool {
	return v.ValidateColumn(column) == nil
}

// FilterValidColumns filters a list of columns, returning only valid ones
// Logs warnings for any invalid columns
func (v *ColumnValidator) FilterValidColumns(columns []string) []string {
	if len(columns) == 0 {
		return columns
	}

	validColumns := make([]string, 0, len(columns))
	for _, col := range columns {
		if v.IsValidColumn(col) {
			validColumns = append(validColumns, col)
		} else {
			logger.Warn("Invalid column '%s' filtered out: column does not exist in model", col)
		}
	}
	return validColumns
}

// ValidateColumns validates multiple column names
// Returns error with details about all invalid columns
func (v *ColumnValidator) ValidateColumns(columns []string) error {
	var invalidColumns []string

	for _, column := range columns {
		if err := v.ValidateColumn(column); err != nil {
			invalidColumns = append(invalidColumns, column)
		}
	}

	if len(invalidColumns) > 0 {
		return fmt.Errorf("invalid columns: %s", strings.Join(invalidColumns, ", "))
	}

	return nil
}

// ValidateRequestOptions validates all column references in RequestOptions
func (v *ColumnValidator) ValidateRequestOptions(options RequestOptions) error {
	// Validate Columns
	if err := v.ValidateColumns(options.Columns); err != nil {
		return fmt.Errorf("in select columns: %w", err)
	}

	// Validate OmitColumns
	if err := v.ValidateColumns(options.OmitColumns); err != nil {
		return fmt.Errorf("in omit columns: %w", err)
	}

	// Validate Filter columns
	for _, filter := range options.Filters {
		if err := v.ValidateColumn(filter.Column); err != nil {
			return fmt.Errorf("in filter: %w", err)
		}
	}

	// Validate Sort columns
	for _, sort := range options.Sort {
		if err := v.ValidateColumn(sort.Column); err != nil {
			return fmt.Errorf("in sort: %w", err)
		}
	}

	// Validate Preload columns (if specified)
	for _, preload := range options.Preload {
		// Note: We don't validate the relation name itself, as it's a relationship
		// Only validate columns if specified for the preload
		if err := v.ValidateColumns(preload.Columns); err != nil {
			return fmt.Errorf("in preload '%s' columns: %w", preload.Relation, err)
		}
		if err := v.ValidateColumns(preload.OmitColumns); err != nil {
			return fmt.Errorf("in preload '%s' omit columns: %w", preload.Relation, err)
		}

		// Validate filter columns in preload
		for _, filter := range preload.Filters {
			if err := v.ValidateColumn(filter.Column); err != nil {
				return fmt.Errorf("in preload '%s' filter: %w", preload.Relation, err)
			}
		}
	}

	return nil
}

// FilterRequestOptions filters all column references in RequestOptions
// Returns a new RequestOptions with only valid columns, logging warnings for invalid ones
func (v *ColumnValidator) FilterRequestOptions(options RequestOptions) RequestOptions {
	filtered := options

	// Filter Columns
	filtered.Columns = v.FilterValidColumns(options.Columns)

	// Filter OmitColumns
	filtered.OmitColumns = v.FilterValidColumns(options.OmitColumns)

	// Filter Filter columns
	validFilters := make([]FilterOption, 0, len(options.Filters))
	for _, filter := range options.Filters {
		if v.IsValidColumn(filter.Column) {
			validFilters = append(validFilters, filter)
		} else {
			logger.Warn("Invalid column in filter '%s' removed", filter.Column)
		}
	}
	filtered.Filters = validFilters

	// Filter Sort columns
	validSorts := make([]SortOption, 0, len(options.Sort))
	for _, sort := range options.Sort {
		if v.IsValidColumn(sort.Column) {
			validSorts = append(validSorts, sort)
		} else {
			logger.Warn("Invalid column in sort '%s' removed", sort.Column)
		}
	}
	filtered.Sort = validSorts

	// Filter Preload columns
	validPreloads := make([]PreloadOption, 0, len(options.Preload))
	for _, preload := range options.Preload {
		filteredPreload := preload
		filteredPreload.Columns = v.FilterValidColumns(preload.Columns)
		filteredPreload.OmitColumns = v.FilterValidColumns(preload.OmitColumns)

		// Filter preload filters
		validPreloadFilters := make([]FilterOption, 0, len(preload.Filters))
		for _, filter := range preload.Filters {
			if v.IsValidColumn(filter.Column) {
				validPreloadFilters = append(validPreloadFilters, filter)
			} else {
				logger.Warn("Invalid column in preload '%s' filter '%s' removed", preload.Relation, filter.Column)
			}
		}
		filteredPreload.Filters = validPreloadFilters

		validPreloads = append(validPreloads, filteredPreload)
	}
	filtered.Preload = validPreloads

	return filtered
}

// GetValidColumns returns a list of all valid column names for debugging purposes
func (v *ColumnValidator) GetValidColumns() []string {
	columns := make([]string, 0, len(v.validColumns))
	for col := range v.validColumns {
		columns = append(columns, col)
	}
	return columns
}
