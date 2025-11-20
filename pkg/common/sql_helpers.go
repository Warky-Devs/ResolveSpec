package common

import (
	"fmt"
	"strings"

	"github.com/bitechdev/ResolveSpec/pkg/logger"
)

// ValidateAndFixPreloadWhere validates that the WHERE clause for a preload contains
// the relation prefix (alias). If not present, it attempts to add it to column references.
// Returns the fixed WHERE clause and an error if it cannot be safely fixed.
func ValidateAndFixPreloadWhere(where string, relationName string) (string, error) {
	if where == "" {
		return where, nil
	}

	// Check if the relation name is already present in the WHERE clause
	lowerWhere := strings.ToLower(where)
	lowerRelation := strings.ToLower(relationName)

	// Check for patterns like "relation.", "relation ", or just "relation" followed by a dot
	if strings.Contains(lowerWhere, lowerRelation+".") ||
		strings.Contains(lowerWhere, "`"+lowerRelation+"`.") ||
		strings.Contains(lowerWhere, "\""+lowerRelation+"\".") {
		// Relation prefix is already present
		return where, nil
	}

	// If the WHERE clause is complex (contains OR, parentheses, subqueries, etc.),
	// we can't safely auto-fix it - require explicit prefix
	if strings.Contains(lowerWhere, " or ") ||
		strings.Contains(where, "(") ||
		strings.Contains(where, ")") {
		return "", fmt.Errorf("preload WHERE condition must reference the relation '%s' (e.g., '%s.column_name'). Complex WHERE clauses with OR/parentheses must explicitly use the relation prefix", relationName, relationName)
	}

	// Try to add the relation prefix to simple column references
	// This handles basic cases like "column = value" or "column = value AND other_column = value"
	// Split by AND to handle multiple conditions (case-insensitive)
	originalConditions := strings.Split(where, " AND ")

	// If uppercase split didn't work, try lowercase
	if len(originalConditions) == 1 {
		originalConditions = strings.Split(where, " and ")
	}

	fixedConditions := make([]string, 0, len(originalConditions))

	for _, cond := range originalConditions {
		cond = strings.TrimSpace(cond)
		if cond == "" {
			continue
		}

		// Check if this condition already has a table prefix (contains a dot)
		if strings.Contains(cond, ".") {
			fixedConditions = append(fixedConditions, cond)
			continue
		}

		// Check if this is a SQL expression/literal that shouldn't be prefixed
		lowerCond := strings.ToLower(strings.TrimSpace(cond))
		if IsSQLExpression(lowerCond) {
			// Don't prefix SQL expressions like "true", "false", "1=1", etc.
			fixedConditions = append(fixedConditions, cond)
			continue
		}

		// Extract the column name (first identifier before operator)
		columnName := ExtractColumnName(cond)
		if columnName == "" {
			// Can't identify column name, require explicit prefix
			return "", fmt.Errorf("preload WHERE condition must reference the relation '%s' (e.g., '%s.column_name'). Cannot auto-fix condition: %s", relationName, relationName, cond)
		}

		// Add relation prefix to the column name only
		fixedCond := strings.Replace(cond, columnName, relationName+"."+columnName, 1)
		fixedConditions = append(fixedConditions, fixedCond)
	}

	fixedWhere := strings.Join(fixedConditions, " AND ")
	logger.Debug("Auto-fixed preload WHERE clause: '%s' -> '%s'", where, fixedWhere)
	return fixedWhere, nil
}

// IsSQLExpression checks if a condition is a SQL expression that shouldn't be prefixed
func IsSQLExpression(cond string) bool {
	// Common SQL literals and expressions
	sqlLiterals := []string{"true", "false", "null", "1=1", "1 = 1", "0=0", "0 = 0"}
	for _, literal := range sqlLiterals {
		if cond == literal {
			return true
		}
	}
	return false
}

// ExtractColumnName extracts the column name from a WHERE condition
// For example: "status = 'active'" returns "status"
func ExtractColumnName(cond string) string {
	// Common SQL operators
	operators := []string{" = ", " != ", " <> ", " > ", " >= ", " < ", " <= ", " LIKE ", " like ", " IN ", " in ", " IS ", " is "}

	for _, op := range operators {
		if idx := strings.Index(cond, op); idx > 0 {
			columnName := strings.TrimSpace(cond[:idx])
			// Remove quotes if present
			columnName = strings.Trim(columnName, "`\"'")
			return columnName
		}
	}

	// If no operator found, check if it's a simple identifier (for boolean columns)
	parts := strings.Fields(cond)
	if len(parts) > 0 {
		columnName := strings.Trim(parts[0], "`\"'")
		// Check if it's a valid identifier (not a SQL keyword)
		if !IsSQLKeyword(strings.ToLower(columnName)) {
			return columnName
		}
	}

	return ""
}

// IsSQLKeyword checks if a string is a SQL keyword that shouldn't be treated as a column name
func IsSQLKeyword(word string) bool {
	keywords := []string{"select", "from", "where", "and", "or", "not", "in", "is", "null", "true", "false", "like", "between", "exists"}
	for _, kw := range keywords {
		if word == kw {
			return true
		}
	}
	return false
}
