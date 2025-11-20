package reflection

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
			result := ExtractColumnFromGormTag(tt.tag)
			if result != tt.expected {
				t.Errorf("ExtractColumnFromGormTag() = %v, want %v", result, tt.expected)
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
			result := ExtractColumnFromBunTag(tt.tag)
			if result != tt.expected {
				t.Errorf("ExtractColumnFromBunTag() = %v, want %v", result, tt.expected)
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

// Test models with embedded structs

type BaseModel struct {
	ID        int    `bun:"rid_base,pk" json:"id"`
	CreatedAt string `bun:"created_at" json:"created_at"`
}

type AdhocBuffer struct {
	CQL1      string `json:"cql1,omitempty" gorm:"->" bun:",scanonly"`
	CQL2      string `json:"cql2,omitempty" gorm:"->" bun:",scanonly"`
	RowNumber int64  `json:"_rownumber,omitempty" gorm:"-" bun:",scanonly"`
}

type ModelWithEmbedded struct {
	BaseModel
	Name        string `bun:"name" json:"name"`
	Description string `bun:"description" json:"description"`
	AdhocBuffer
}

type GormBaseModel struct {
	ID        int    `gorm:"column:rid_base;primaryKey" json:"id"`
	CreatedAt string `gorm:"column:created_at" json:"created_at"`
}

type GormAdhocBuffer struct {
	CQL1      string `json:"cql1,omitempty" gorm:"column:cql1;->" bun:",scanonly"`
	CQL2      string `json:"cql2,omitempty" gorm:"column:cql2;->" bun:",scanonly"`
	RowNumber int64  `json:"_rownumber,omitempty" gorm:"-" bun:",scanonly"`
}

type GormModelWithEmbedded struct {
	GormBaseModel
	Name        string `gorm:"column:name" json:"name"`
	Description string `gorm:"column:description" json:"description"`
	GormAdhocBuffer
}

func TestGetPrimaryKeyNameWithEmbedded(t *testing.T) {
	tests := []struct {
		name     string
		model    any
		expected string
	}{
		{
			name:     "Bun model with embedded base",
			model:    ModelWithEmbedded{},
			expected: "rid_base",
		},
		{
			name:     "Bun model with embedded base (pointer)",
			model:    &ModelWithEmbedded{},
			expected: "rid_base",
		},
		{
			name:     "GORM model with embedded base",
			model:    GormModelWithEmbedded{},
			expected: "rid_base",
		},
		{
			name:     "GORM model with embedded base (pointer)",
			model:    &GormModelWithEmbedded{},
			expected: "rid_base",
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

func TestGetPrimaryKeyValueWithEmbedded(t *testing.T) {
	bunModel := ModelWithEmbedded{
		BaseModel: BaseModel{
			ID:        123,
			CreatedAt: "2024-01-01",
		},
		Name:        "Test",
		Description: "Test Description",
	}

	gormModel := GormModelWithEmbedded{
		GormBaseModel: GormBaseModel{
			ID:        456,
			CreatedAt: "2024-01-02",
		},
		Name:        "GORM Test",
		Description: "GORM Test Description",
	}

	tests := []struct {
		name     string
		model    any
		expected any
	}{
		{
			name:     "Bun model with embedded base",
			model:    bunModel,
			expected: 123,
		},
		{
			name:     "Bun model with embedded base (pointer)",
			model:    &bunModel,
			expected: 123,
		},
		{
			name:     "GORM model with embedded base",
			model:    gormModel,
			expected: 456,
		},
		{
			name:     "GORM model with embedded base (pointer)",
			model:    &gormModel,
			expected: 456,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPrimaryKeyValue(tt.model)
			if result != tt.expected {
				t.Errorf("GetPrimaryKeyValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetModelColumnsWithEmbedded(t *testing.T) {
	tests := []struct {
		name     string
		model    any
		expected []string
	}{
		{
			name:     "Bun model with embedded structs",
			model:    ModelWithEmbedded{},
			expected: []string{"rid_base", "created_at", "name", "description", "cql1", "cql2", "_rownumber"},
		},
		{
			name:     "Bun model with embedded structs (pointer)",
			model:    &ModelWithEmbedded{},
			expected: []string{"rid_base", "created_at", "name", "description", "cql1", "cql2", "_rownumber"},
		},
		{
			name:     "GORM model with embedded structs",
			model:    GormModelWithEmbedded{},
			expected: []string{"rid_base", "created_at", "name", "description", "cql1", "cql2", "_rownumber"},
		},
		{
			name:     "GORM model with embedded structs (pointer)",
			model:    &GormModelWithEmbedded{},
			expected: []string{"rid_base", "created_at", "name", "description", "cql1", "cql2", "_rownumber"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetModelColumns(tt.model)
			if len(result) != len(tt.expected) {
				t.Errorf("GetModelColumns() returned %d columns, want %d. Got: %v", len(result), len(tt.expected), result)
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

func TestIsColumnWritableWithEmbedded(t *testing.T) {
	tests := []struct {
		name       string
		model      any
		columnName string
		expected   bool
	}{
		{
			name:       "Bun model - writable column in main struct",
			model:      ModelWithEmbedded{},
			columnName: "name",
			expected:   true,
		},
		{
			name:       "Bun model - writable column in embedded base",
			model:      ModelWithEmbedded{},
			columnName: "rid_base",
			expected:   true,
		},
		{
			name:       "Bun model - scanonly column in embedded adhoc buffer",
			model:      ModelWithEmbedded{},
			columnName: "cql1",
			expected:   false,
		},
		{
			name:       "Bun model - scanonly column _rownumber",
			model:      ModelWithEmbedded{},
			columnName: "_rownumber",
			expected:   false,
		},
		{
			name:       "GORM model - writable column in main struct",
			model:      GormModelWithEmbedded{},
			columnName: "name",
			expected:   true,
		},
		{
			name:       "GORM model - writable column in embedded base",
			model:      GormModelWithEmbedded{},
			columnName: "rid_base",
			expected:   true,
		},
		{
			name:       "GORM model - readonly column in embedded adhoc buffer",
			model:      GormModelWithEmbedded{},
			columnName: "cql1",
			expected:   false,
		},
		{
			name:       "GORM model - readonly column _rownumber",
			model:      GormModelWithEmbedded{},
			columnName: "_rownumber",
			expected:   false, // bun:",scanonly" marks it as read-only, takes precedence over gorm:"-"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsColumnWritable(tt.model, tt.columnName)
			if result != tt.expected {
				t.Errorf("IsColumnWritable(%s) = %v, want %v", tt.columnName, result, tt.expected)
			}
		})
	}
}
