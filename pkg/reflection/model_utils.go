package reflection

import (
	"reflect"
	"strings"

	"github.com/bitechdev/ResolveSpec/pkg/modelregistry"
)

type PrimaryKeyNameProvider interface {
	GetIDName() string
}

// GetPrimaryKeyName extracts the primary key column name from a model
// It first checks if the model implements PrimaryKeyNameProvider (GetIDName method)
// Falls back to reflection to find bun:",pk" tag, then gorm:"primaryKey" tag
func GetPrimaryKeyName(model any) string {
	if reflect.TypeOf(model) == nil {
		return ""
	}
	// If we are given a string model name, look up the model
	if reflect.TypeOf(model).Kind() == reflect.String {
		name := model.(string)
		m, err := modelregistry.GetModelByName(name)
		if err == nil {
			model = m
		}
	}

	// Check if model implements PrimaryKeyNameProvider
	if provider, ok := model.(PrimaryKeyNameProvider); ok {
		return provider.GetIDName()
	}

	// Try Bun tag first
	if pkName := getPrimaryKeyFromReflection(model, "bun"); pkName != "" {
		return pkName
	}

	// Fall back to GORM tag
	if pkName := getPrimaryKeyFromReflection(model, "gorm"); pkName != "" {
		return pkName
	}

	return ""
}

// GetPrimaryKeyValue extracts the primary key value from a model instance
// Returns the value of the primary key field
func GetPrimaryKeyValue(model any) interface{} {
	if model == nil || reflect.TypeOf(model) == nil {
		return nil
	}

	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()

	// Try Bun tag first
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		bunTag := field.Tag.Get("bun")
		if strings.Contains(bunTag, "pk") {
			fieldValue := val.Field(i)
			if fieldValue.CanInterface() {
				return fieldValue.Interface()
			}
		}
	}

	// Fall back to GORM tag
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		gormTag := field.Tag.Get("gorm")
		if strings.Contains(gormTag, "primaryKey") {
			fieldValue := val.Field(i)
			if fieldValue.CanInterface() {
				return fieldValue.Interface()
			}
		}
	}

	// Last resort: look for field named "ID" or "Id"
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if strings.ToLower(field.Name) == "id" {
			fieldValue := val.Field(i)
			if fieldValue.CanInterface() {
				return fieldValue.Interface()
			}
		}
	}

	return nil
}

// GetModelColumns extracts all column names from a model using reflection
// It checks bun tags first, then gorm tags, then json tags, and finally falls back to lowercase field names
func GetModelColumns(model any) []string {
	var columns []string

	modelType := reflect.TypeOf(model)

	// Unwrap pointers, slices, and arrays to get to the base struct type
	for modelType != nil && (modelType.Kind() == reflect.Pointer || modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array) {
		modelType = modelType.Elem()
	}

	// Validate that we have a struct type
	if modelType == nil || modelType.Kind() != reflect.Struct {
		return columns
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Get column name using the same logic as primary key extraction
		columnName := getColumnNameFromField(field)

		if columnName != "" {
			columns = append(columns, columnName)
		}
	}

	return columns
}

// getColumnNameFromField extracts the column name from a struct field
// Priority: bun tag -> gorm tag -> json tag -> lowercase field name
func getColumnNameFromField(field reflect.StructField) string {
	// Try bun tag first
	bunTag := field.Tag.Get("bun")
	if bunTag != "" && bunTag != "-" {
		if colName := ExtractColumnFromBunTag(bunTag); colName != "" {
			return colName
		}
	}

	// Try gorm tag
	gormTag := field.Tag.Get("gorm")
	if gormTag != "" && gormTag != "-" {
		if colName := ExtractColumnFromGormTag(gormTag); colName != "" {
			return colName
		}
	}

	// Fall back to json tag
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		// Extract just the field name before any options
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Last resort: use field name in lowercase
	return strings.ToLower(field.Name)
}

// getPrimaryKeyFromReflection uses reflection to find the primary key field
func getPrimaryKeyFromReflection(model any, ormType string) string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ""
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		switch ormType {
		case "gorm":
			// Check for gorm tag with primaryKey
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "primaryKey") {
				// Try to extract column name from gorm tag
				if colName := ExtractColumnFromGormTag(gormTag); colName != "" {
					return colName
				}
				// Fall back to json tag
				if jsonTag := field.Tag.Get("json"); jsonTag != "" {
					return strings.Split(jsonTag, ",")[0]
				}
			}
		case "bun":
			// Check for bun tag with pk flag
			bunTag := field.Tag.Get("bun")
			if strings.Contains(bunTag, "pk") {
				// Extract column name from bun tag
				if colName := ExtractColumnFromBunTag(bunTag); colName != "" {
					return colName
				}
				// Fall back to json tag
				if jsonTag := field.Tag.Get("json"); jsonTag != "" {
					return strings.Split(jsonTag, ",")[0]
				}
			}
		}
	}

	return ""
}

// ExtractColumnFromGormTag extracts the column name from a gorm tag
// Example: "column:id;primaryKey" -> "id"
func ExtractColumnFromGormTag(tag string) string {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if colName, found := strings.CutPrefix(part, "column:"); found {
			return colName
		}
	}
	return ""
}

// ExtractColumnFromBunTag extracts the column name from a bun tag
// Example: "id,pk" -> "id"
// Example: ",pk" -> "" (will fall back to json tag)
func ExtractColumnFromBunTag(tag string) string {
	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

// IsColumnWritable checks if a column can be written to in the database
// For bun: returns false if the field has "scanonly" tag
// For gorm: returns false if the field has "<-:false" or "->" (read-only) tag
func IsColumnWritable(model any, columnName string) bool {
	modelType := reflect.TypeOf(model)

	// Unwrap pointers to get to the base struct type
	for modelType != nil && modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}

	// Validate that we have a struct type
	if modelType == nil || modelType.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Check if this field matches the column name
		fieldColumnName := getColumnNameFromField(field)
		if fieldColumnName != columnName {
			continue
		}

		// Check bun tag for scanonly
		bunTag := field.Tag.Get("bun")
		if bunTag != "" {
			if isBunFieldScanOnly(bunTag) {
				return false
			}
		}

		// Check gorm tag for write restrictions
		gormTag := field.Tag.Get("gorm")
		if gormTag != "" {
			if isGormFieldReadOnly(gormTag) {
				return false
			}
		}

		// Column is writable
		return true
	}

	// Column not found in model, allow it (might be a dynamic column)
	return true
}

// isBunFieldScanOnly checks if a bun tag indicates the field is scan-only
// Example: "column_name,scanonly" -> true
func isBunFieldScanOnly(tag string) bool {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		if strings.TrimSpace(part) == "scanonly" {
			return true
		}
	}
	return false
}

// isGormFieldReadOnly checks if a gorm tag indicates the field is read-only
// Examples:
//   - "<-:false" -> true (no writes allowed)
//   - "->" -> true (read-only, common pattern)
//   - "column:name;->" -> true
//   - "<-:create" -> false (writes allowed on create)
func isGormFieldReadOnly(tag string) bool {
	parts := strings.Split(tag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check for read-only marker
		if part == "->" {
			return true
		}

		// Check for write restrictions
		if value, found := strings.CutPrefix(part, "<-:"); found {
			if value == "false" {
				return true
			}
		}
	}
	return false
}
