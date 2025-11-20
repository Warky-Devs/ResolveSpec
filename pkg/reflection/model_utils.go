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
func GetPrimaryKeyValue(model any) any {
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

	// Try Bun tag first
	if pkValue := findPrimaryKeyValue(val, "bun"); pkValue != nil {
		return pkValue
	}

	// Fall back to GORM tag
	if pkValue := findPrimaryKeyValue(val, "gorm"); pkValue != nil {
		return pkValue
	}

	// Last resort: look for field named "ID" or "Id"
	if pkValue := findFieldByName(val, "id"); pkValue != nil {
		return pkValue
	}

	return nil
}

// findPrimaryKeyValue recursively searches for a primary key field in the struct
func findPrimaryKeyValue(val reflect.Value, ormType string) any {
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Check if this is an embedded struct
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// Recursively search in embedded struct
			if pkValue := findPrimaryKeyValue(fieldValue, ormType); pkValue != nil {
				return pkValue
			}
			continue
		}

		// Check for primary key tag
		switch ormType {
		case "bun":
			bunTag := field.Tag.Get("bun")
			if strings.Contains(bunTag, "pk") && fieldValue.CanInterface() {
				return fieldValue.Interface()
			}
		case "gorm":
			gormTag := field.Tag.Get("gorm")
			if strings.Contains(gormTag, "primaryKey") && fieldValue.CanInterface() {
				return fieldValue.Interface()
			}
		}
	}

	return nil
}

// findFieldByName recursively searches for a field by name in the struct
func findFieldByName(val reflect.Value, name string) any {
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Check if this is an embedded struct
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			// Recursively search in embedded struct
			if result := findFieldByName(fieldValue, name); result != nil {
				return result
			}
			continue
		}

		// Check if field name matches
		if strings.ToLower(field.Name) == name && fieldValue.CanInterface() {
			return fieldValue.Interface()
		}
	}

	return nil
}

// GetModelColumns extracts all column names from a model using reflection
// It checks bun tags first, then gorm tags, then json tags, and finally falls back to lowercase field names
// This function recursively processes embedded structs to include their fields
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

	collectColumnsFromType(modelType, &columns)

	return columns
}

// collectColumnsFromType recursively collects column names from a struct type and its embedded fields
func collectColumnsFromType(typ reflect.Type, columns *[]string) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check if this is an embedded struct
		if field.Anonymous {
			// Unwrap pointer type if necessary
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}

			// Recursively process embedded struct
			if fieldType.Kind() == reflect.Struct {
				collectColumnsFromType(fieldType, columns)
				continue
			}
		}

		// Get column name using the same logic as primary key extraction
		columnName := getColumnNameFromField(field)

		if columnName != "" {
			*columns = append(*columns, columnName)
		}
	}
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
// This function recursively searches embedded structs
func getPrimaryKeyFromReflection(model any, ormType string) string {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ""
	}

	typ := val.Type()
	return findPrimaryKeyNameFromType(typ, ormType)
}

// findPrimaryKeyNameFromType recursively searches for the primary key field name in a struct type
func findPrimaryKeyNameFromType(typ reflect.Type, ormType string) string {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check if this is an embedded struct
		if field.Anonymous {
			// Unwrap pointer type if necessary
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}

			// Recursively search in embedded struct
			if fieldType.Kind() == reflect.Struct {
				if pkName := findPrimaryKeyNameFromType(fieldType, ormType); pkName != "" {
					return pkName
				}
			}
			continue
		}

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
	if strings.HasPrefix(strings.ToLower(tag), "table:") || strings.HasPrefix(strings.ToLower(tag), "rel:") || strings.HasPrefix(strings.ToLower(tag), "join:") {
		return ""
	}
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}

// IsColumnWritable checks if a column can be written to in the database
// For bun: returns false if the field has "scanonly" tag
// For gorm: returns false if the field has "<-:false" or "->" (read-only) tag
// This function recursively searches embedded structs
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

	found, writable := isColumnWritableInType(modelType, columnName)
	if found {
		return writable
	}

	// Column not found in model, allow it (might be a dynamic column)
	return true
}

// isColumnWritableInType recursively searches for a column and checks if it's writable
// Returns (found, writable) where found indicates if the column was found
func isColumnWritableInType(typ reflect.Type, columnName string) (bool, bool) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check if this is an embedded struct
		if field.Anonymous {
			// Unwrap pointer type if necessary
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}

			// Recursively search in embedded struct
			if fieldType.Kind() == reflect.Struct {
				if found, writable := isColumnWritableInType(fieldType, columnName); found {
					return true, writable
				}
			}
			continue
		}

		// Check if this field matches the column name
		fieldColumnName := getColumnNameFromField(field)
		if fieldColumnName != columnName {
			continue
		}

		// Found the field, now check if it's writable
		// Check bun tag for scanonly
		bunTag := field.Tag.Get("bun")
		if bunTag != "" {
			if isBunFieldScanOnly(bunTag) {
				return true, false
			}
		}

		// Check gorm tag for write restrictions
		gormTag := field.Tag.Get("gorm")
		if gormTag != "" {
			if isGormFieldReadOnly(gormTag) {
				return true, false
			}
		}

		// Column is writable
		return true, true
	}

	// Column not found
	return false, false
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
