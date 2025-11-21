package common

import (
	"testing"

	"github.com/bitechdev/ResolveSpec/pkg/modelregistry"
)

func TestSanitizeWhereClause(t *testing.T) {
	tests := []struct {
		name      string
		where     string
		tableName string
		expected  string
	}{
		{
			name:      "trivial conditions in parentheses",
			where:     "(true AND true AND true)",
			tableName: "mastertask",
			expected:  "",
		},
		{
			name:      "trivial conditions without parentheses",
			where:     "true AND true AND true",
			tableName: "mastertask",
			expected:  "",
		},
		{
			name:      "single trivial condition",
			where:     "true",
			tableName: "mastertask",
			expected:  "",
		},
		{
			name:      "valid condition with parentheses",
			where:     "(status = 'active')",
			tableName: "users",
			expected:  "users.status = 'active'",
		},
		{
			name:      "mixed trivial and valid conditions",
			where:     "true AND status = 'active' AND 1=1",
			tableName: "users",
			expected:  "users.status = 'active'",
		},
		{
			name:      "condition already with table prefix",
			where:     "users.status = 'active'",
			tableName: "users",
			expected:  "users.status = 'active'",
		},
		{
			name:      "multiple valid conditions",
			where:     "status = 'active' AND age > 18",
			tableName: "users",
			expected:  "users.status = 'active' AND users.age > 18",
		},
		{
			name:      "no table name provided",
			where:     "status = 'active'",
			tableName: "",
			expected:  "status = 'active'",
		},
		{
			name:      "empty where clause",
			where:     "",
			tableName: "users",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeWhereClause(tt.where, tt.tableName)
			if result != tt.expected {
				t.Errorf("SanitizeWhereClause(%q, %q) = %q; want %q", tt.where, tt.tableName, result, tt.expected)
			}
		})
	}
}

func TestStripOuterParentheses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single level parentheses",
			input:    "(true)",
			expected: "true",
		},
		{
			name:     "multiple levels",
			input:    "((true))",
			expected: "true",
		},
		{
			name:     "no parentheses",
			input:    "true",
			expected: "true",
		},
		{
			name:     "mismatched parentheses",
			input:    "(true",
			expected: "(true",
		},
		{
			name:     "complex expression",
			input:    "(a AND b)",
			expected: "a AND b",
		},
		{
			name:     "nested but not outer",
			input:    "(a AND (b OR c)) AND d",
			expected: "(a AND (b OR c)) AND d",
		},
		{
			name:     "with spaces",
			input:    "  ( true )  ",
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripOuterParentheses(tt.input)
			if result != tt.expected {
				t.Errorf("stripOuterParentheses(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsTrivialCondition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true", "true", true},
		{"true with spaces", "  true  ", true},
		{"TRUE uppercase", "TRUE", true},
		{"1=1", "1=1", true},
		{"1 = 1", "1 = 1", true},
		{"true = true", "true = true", true},
		{"valid condition", "status = 'active'", false},
		{"false", "false", false},
		{"column name", "is_active", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTrivialCondition(tt.input)
			if result != tt.expected {
				t.Errorf("IsTrivialCondition(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test model for model-aware sanitization tests
type MasterTask struct {
	ID     int    `bun:"id,pk"`
	Name   string `bun:"name"`
	Status string `bun:"status"`
	UserID int    `bun:"user_id"`
}

func TestSanitizeWhereClauseWithModel(t *testing.T) {
	// Register the test model
	err := modelregistry.RegisterModel(MasterTask{}, "mastertask")
	if err != nil {
		// Model might already be registered, ignore error
		t.Logf("Model registration returned: %v", err)
	}

	tests := []struct {
		name      string
		where     string
		tableName string
		expected  string
	}{
		{
			name:      "valid column gets prefixed",
			where:     "status = 'active'",
			tableName: "mastertask",
			expected:  "mastertask.status = 'active'",
		},
		{
			name:      "multiple valid columns get prefixed",
			where:     "status = 'active' AND user_id = 123",
			tableName: "mastertask",
			expected:  "mastertask.status = 'active' AND mastertask.user_id = 123",
		},
		{
			name:      "invalid column does not get prefixed",
			where:     "invalid_column = 'value'",
			tableName: "mastertask",
			expected:  "invalid_column = 'value'",
		},
		{
			name:      "mix of valid and trivial conditions",
			where:     "true AND status = 'active' AND 1=1",
			tableName: "mastertask",
			expected:  "mastertask.status = 'active'",
		},
		{
			name:      "parentheses with valid column",
			where:     "(status = 'active')",
			tableName: "mastertask",
			expected:  "mastertask.status = 'active'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeWhereClause(tt.where, tt.tableName)
			if result != tt.expected {
				t.Errorf("SanitizeWhereClause(%q, %q) = %q; want %q", tt.where, tt.tableName, result, tt.expected)
			}
		})
	}
}
