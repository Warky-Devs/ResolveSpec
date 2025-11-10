package database

import (
	"strings"
)

// parseTableName splits a table name that may contain schema into separate schema and table
// For example: "public.users" -> ("public", "users")
//
//	"users" -> ("", "users")
func parseTableName(fullTableName string) (schema, table string) {
	if idx := strings.LastIndex(fullTableName, "."); idx != -1 {
		return fullTableName[:idx], fullTableName[idx+1:]
	}
	return "", fullTableName
}
