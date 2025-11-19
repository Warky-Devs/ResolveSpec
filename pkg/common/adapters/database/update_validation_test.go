package database

import (
	"testing"

	"github.com/bitechdev/ResolveSpec/pkg/reflection"
)

// Test models for bun
type BunTestModel struct {
	ID          int    `bun:"id,pk"`
	Name        string `bun:"name"`
	Email       string `bun:"email"`
	ComputedCol string `bun:"computed_col,scanonly"`
}

// Test models for gorm
type GormTestModel struct {
	ID          int    `gorm:"column:id;primaryKey"`
	Name        string `gorm:"column:name"`
	Email       string `gorm:"column:email"`
	ReadOnlyCol string `gorm:"column:readonly_col;->"`
	NoWriteCol  string `gorm:"column:nowrite_col;<-:false"`
}

func TestIsColumnWritable_Bun(t *testing.T) {
	model := &BunTestModel{}

	tests := []struct {
		name       string
		columnName string
		expected   bool
	}{
		{
			name:       "writable column - id",
			columnName: "id",
			expected:   true,
		},
		{
			name:       "writable column - name",
			columnName: "name",
			expected:   true,
		},
		{
			name:       "writable column - email",
			columnName: "email",
			expected:   true,
		},
		{
			name:       "scanonly column should not be writable",
			columnName: "computed_col",
			expected:   false,
		},
		{
			name:       "non-existent column should be writable (dynamic)",
			columnName: "nonexistent",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reflection.IsColumnWritable(model, tt.columnName)
			if result != tt.expected {
				t.Errorf("IsColumnWritable(%q) = %v, want %v", tt.columnName, result, tt.expected)
			}
		})
	}
}

func TestIsColumnWritable_Gorm(t *testing.T) {
	model := &GormTestModel{}

	tests := []struct {
		name       string
		columnName string
		expected   bool
	}{
		{
			name:       "writable column - id",
			columnName: "id",
			expected:   true,
		},
		{
			name:       "writable column - name",
			columnName: "name",
			expected:   true,
		},
		{
			name:       "writable column - email",
			columnName: "email",
			expected:   true,
		},
		{
			name:       "read-only column with -> should not be writable",
			columnName: "readonly_col",
			expected:   false,
		},
		{
			name:       "column with <-:false should not be writable",
			columnName: "nowrite_col",
			expected:   false,
		},
		{
			name:       "non-existent column should be writable (dynamic)",
			columnName: "nonexistent",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reflection.IsColumnWritable(model, tt.columnName)
			if result != tt.expected {
				t.Errorf("IsColumnWritable(%q) = %v, want %v", tt.columnName, result, tt.expected)
			}
		})
	}
}

func TestBunUpdateQuery_SetMap_FiltersScanOnly(t *testing.T) {
	// Note: This is a unit test for the validation logic only.
	// We can't fully test the bun query without a database connection,
	// but we've verified the validation logic in TestIsColumnWritable_Bun
	t.Skip("Skipping integration test - validation logic tested in TestIsColumnWritable_Bun")
}

func TestGormUpdateQuery_SetMap_FiltersReadOnly(t *testing.T) {
	model := &GormTestModel{}
	query := &GormUpdateQuery{
		model: model,
	}

	// SetMap should filter out read-only columns
	values := map[string]interface{}{
		"name":         "John",
		"email":        "john@example.com",
		"readonly_col": "should_be_filtered",
		"nowrite_col":  "should_also_be_filtered",
	}

	query.SetMap(values)

	// Check that the updates map only contains writable columns
	if updates, ok := query.updates.(map[string]interface{}); ok {
		if _, exists := updates["readonly_col"]; exists {
			t.Error("readonly_col should have been filtered out")
		}
		if _, exists := updates["nowrite_col"]; exists {
			t.Error("nowrite_col should have been filtered out")
		}
		if _, exists := updates["name"]; !exists {
			t.Error("name should be in updates")
		}
		if _, exists := updates["email"]; !exists {
			t.Error("email should be in updates")
		}
	} else {
		t.Error("updates should be a map[string]interface{}")
	}
}
