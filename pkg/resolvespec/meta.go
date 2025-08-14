package resolvespec

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/Warky-Devs/ResolveSpec/pkg/logger"
)

func (h *LegacyAPIHandler) HandleGet(w http.ResponseWriter, r *http.Request, params map[string]string) {
	schema := params["schema"]
	entity := params["entity"]

	logger.Info("Getting metadata for %s.%s", schema, entity)

	// Get model for the entity
	model, err := h.getModelForEntity(schema, entity)
	if err != nil {
		logger.Error("Failed to get model: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_entity", "Invalid entity", err)
		return
	}

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

	// Get field information using reflection
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Parse GORM tags
		gormTag := field.Tag.Get("gorm")
		jsonTag := field.Tag.Get("json")

		// Skip if json tag is "-"
		if jsonTag == "-" {
			continue
		}

		// Get JSON field name
		jsonName := strings.Split(jsonTag, ",")[0]
		if jsonName == "" {
			jsonName = field.Name
		}

		// Check if it's a relation
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

	h.sendResponse(w, metadata, nil)
}

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
