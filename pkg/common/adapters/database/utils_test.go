package database

import (
	"testing"
)

// Test models for GORM
type GormModelWithGetIDName struct {
	ID   int    `gorm:"column:rid_test;primaryKey" json:"id"`
	Name string `json:"name"`
}

func (m GormModelWithGetIDName) GetIDName() string {
	return "rid_test"
}

type GormModelWithColumnTag struct {
	ID   int    `gorm:"column:custom_id;primaryKey" json:"id"`
	Name string `json:"name"`
}

type GormModelWithJSONFallback struct {
	ID   int    `gorm:"primaryKey" json:"user_id"`
	Name string `json:"name"`
}

// Test models for Bun
type BunModelWithGetIDName struct {
	ID   int    `bun:"rid_test,pk" json:"id"`
	Name string `json:"name"`
}

func (m BunModelWithGetIDName) GetIDName() string {
	return "rid_test"
}

type BunModelWithColumnTag struct {
	ID   int    `bun:"custom_id,pk" json:"id"`
	Name string `json:"name"`
}

type BunModelWithJSONFallback struct {
	ID   int    `bun:",pk" json:"user_id"`
	Name string `json:"name"`
}

func TestGetPrimaryKeyName(t *testing.T) {
	tests := []struct {
		name     string
		model    any
		expected string
	}{
		{
			name:     "GORM model with GetIDName method",
			model:    GormModelWithGetIDName{},
			expected: "rid_test",
		},
		{
			name:     "GORM model with column tag",
			model:    GormModelWithColumnTag{},
			expected: "custom_id",
		},
		{
			name:     "GORM model with JSON fallback",
			model:    GormModelWithJSONFallback{},
			expected: "user_id",
		},
		{
			name:     "GORM model pointer with GetIDName",
			model:    &GormModelWithGetIDName{},
			expected: "rid_test",
		},
		{
			name:     "GORM model pointer with column tag",
			model:    &GormModelWithColumnTag{},
			expected: "custom_id",
		},
		{
			name:     "Bun model with GetIDName method",
			model:    BunModelWithGetIDName{},
			expected: "rid_test",
		},
		{
			name:     "Bun model with column tag",
			model:    BunModelWithColumnTag{},
			expected: "custom_id",
		},
		{
			name:     "Bun model with JSON fallback",
			model:    BunModelWithJSONFallback{},
			expected: "user_id",
		},
		{
			name:     "Bun model pointer with GetIDName",
			model:    &BunModelWithGetIDName{},
			expected: "rid_test",
		},
		{
			name:     "Bun model pointer with column tag",
			model:    &BunModelWithColumnTag{},
			expected: "custom_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrimaryKeyName(tt.model)
			if result != tt.expected {
				t.Errorf("GetPrimaryKeyName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractColumnFromGormTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "column tag with primaryKey",
			tag:      "column:rid_test;primaryKey",
			expected: "rid_test",
		},
		{
			name:     "column tag with spaces",
			tag:      "column:user_id ; primaryKey ; autoIncrement",
			expected: "user_id",
		},
		{
			name:     "no column tag",
			tag:      "primaryKey;autoIncrement",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractColumnFromGormTag(tt.tag)
			if result != tt.expected {
				t.Errorf("extractColumnFromGormTag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractColumnFromBunTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "column name with pk flag",
			tag:      "rid_test,pk",
			expected: "rid_test",
		},
		{
			name:     "only pk flag",
			tag:      ",pk",
			expected: "",
		},
		{
			name:     "column with multiple flags",
			tag:      "user_id,pk,autoincrement",
			expected: "user_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractColumnFromBunTag(tt.tag)
			if result != tt.expected {
				t.Errorf("extractColumnFromBunTag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetModelColumns(t *testing.T) {
	tests := []struct {
		name     string
		model    any
		expected []string
	}{
		{
			name:     "Bun model with multiple columns",
			model:    BunModelWithColumnTag{},
			expected: []string{"custom_id", "name"},
		},
		{
			name:     "GORM model with multiple columns",
			model:    GormModelWithColumnTag{},
			expected: []string{"custom_id", "name"},
		},
		{
			name:     "Bun model pointer",
			model:    &BunModelWithColumnTag{},
			expected: []string{"custom_id", "name"},
		},
		{
			name:     "GORM model pointer",
			model:    &GormModelWithColumnTag{},
			expected: []string{"custom_id", "name"},
		},
		{
			name:     "Bun model with JSON fallback",
			model:    BunModelWithJSONFallback{},
			expected: []string{"user_id", "name"},
		},
		{
			name:     "GORM model with JSON fallback",
			model:    GormModelWithJSONFallback{},
			expected: []string{"user_id", "name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetModelColumns(tt.model)
			if len(result) != len(tt.expected) {
				t.Errorf("GetModelColumns() returned %d columns, want %d", len(result), len(tt.expected))
				return
			}
			for i, col := range result {
				if col != tt.expected[i] {
					t.Errorf("GetModelColumns()[%d] = %v, want %v", i, col, tt.expected[i])
				}
			}
		})
	}
}
