package restheadspec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModel represents a typical model with RowNumber field (like DBAdhocBuffer)
type TestModel struct {
	ID        int64  `json:"id" bun:"id,pk"`
	Name      string `json:"name" bun:"name"`
	RowNumber int64  `json:"_rownumber,omitempty" gorm:"-" bun:"-"`
}

func TestSetRowNumbersOnRecords(t *testing.T) {
	handler := &Handler{}

	tests := []struct {
		name     string
		records  any
		offset   int
		expected []int64
	}{
		{
			name: "Set row numbers on slice of pointers",
			records: []*TestModel{
				{ID: 1, Name: "First"},
				{ID: 2, Name: "Second"},
				{ID: 3, Name: "Third"},
			},
			offset:   0,
			expected: []int64{1, 2, 3},
		},
		{
			name: "Set row numbers with offset",
			records: []*TestModel{
				{ID: 11, Name: "Eleventh"},
				{ID: 12, Name: "Twelfth"},
			},
			offset:   10,
			expected: []int64{11, 12},
		},
		{
			name: "Set row numbers on slice of values",
			records: []TestModel{
				{ID: 1, Name: "First"},
				{ID: 2, Name: "Second"},
			},
			offset:   5,
			expected: []int64{6, 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.setRowNumbersOnRecords(tt.records, tt.offset)

			// Verify row numbers were set correctly
			switch records := tt.records.(type) {
			case []*TestModel:
				assert.Equal(t, len(tt.expected), len(records))
				for i, record := range records {
					assert.Equal(t, tt.expected[i], record.RowNumber,
						"Record %d should have RowNumber=%d", i, tt.expected[i])
				}
			case []TestModel:
				assert.Equal(t, len(tt.expected), len(records))
				for i, record := range records {
					assert.Equal(t, tt.expected[i], record.RowNumber,
						"Record %d should have RowNumber=%d", i, tt.expected[i])
				}
			}
		})
	}
}

func TestSetRowNumbersOnRecords_NoRowNumberField(t *testing.T) {
	handler := &Handler{}

	// Model without RowNumber field
	type SimpleModel struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	records := []*SimpleModel{
		{ID: 1, Name: "First"},
		{ID: 2, Name: "Second"},
	}

	// Should not panic when model doesn't have RowNumber field
	assert.NotPanics(t, func() {
		handler.setRowNumbersOnRecords(records, 0)
	})
}

func TestSetRowNumbersOnRecords_NilRecords(t *testing.T) {
	handler := &Handler{}

	records := []*TestModel{
		{ID: 1, Name: "First"},
		nil, // Nil record
		{ID: 3, Name: "Third"},
	}

	// Should not panic with nil records
	assert.NotPanics(t, func() {
		handler.setRowNumbersOnRecords(records, 0)
	})

	// Verify non-nil records were set
	assert.Equal(t, int64(1), records[0].RowNumber)
	assert.Equal(t, int64(3), records[2].RowNumber)
}

// DBAdhocBuffer simulates the actual DBAdhocBuffer from db package
type DBAdhocBuffer struct {
	CQL1      string `json:"cql1,omitempty" gorm:"->" bun:"-"`
	RowNumber int64  `json:"_rownumber,omitempty" gorm:"-" bun:"-"`
}

// ModelWithEmbeddedBuffer simulates a real model like ModelPublicConsultant
type ModelWithEmbeddedBuffer struct {
	ID   int64  `json:"id" bun:"id,pk"`
	Name string `json:"name" bun:"name"`

	DBAdhocBuffer `json:",omitempty"` // Embedded struct containing RowNumber
}

func TestSetRowNumbersOnRecords_EmbeddedBuffer(t *testing.T) {
	handler := &Handler{}

	// Test with embedded DBAdhocBuffer (like real models)
	records := []*ModelWithEmbeddedBuffer{
		{ID: 1, Name: "First"},
		{ID: 2, Name: "Second"},
		{ID: 3, Name: "Third"},
	}

	handler.setRowNumbersOnRecords(records, 10)

	// Verify row numbers were set on embedded field
	assert.Equal(t, int64(11), records[0].RowNumber, "First record should have RowNumber=11")
	assert.Equal(t, int64(12), records[1].RowNumber, "Second record should have RowNumber=12")
	assert.Equal(t, int64(13), records[2].RowNumber, "Third record should have RowNumber=13")
}

func TestSetRowNumbersOnRecords_EmbeddedBuffer_SliceOfValues(t *testing.T) {
	handler := &Handler{}

	// Test with slice of values (not pointers)
	records := []ModelWithEmbeddedBuffer{
		{ID: 1, Name: "First"},
		{ID: 2, Name: "Second"},
	}

	handler.setRowNumbersOnRecords(records, 0)

	// Verify row numbers were set on embedded field
	assert.Equal(t, int64(1), records[0].RowNumber, "First record should have RowNumber=1")
	assert.Equal(t, int64(2), records[1].RowNumber, "Second record should have RowNumber=2")
}

// Simulate the exact structure from user's code
type MockDBAdhocBuffer struct {
	CQL1      string `json:"cql1,omitempty" gorm:"->" bun:"-"`
	CQL2      string `json:"cql2,omitempty" gorm:"->" bun:"-"`
	RowNumber int64  `json:"_rownumber,omitempty" gorm:"-" bun:"-"`
	Request   string `json:"_request,omitempty" gorm:"-" bun:"-"`
}

// Exact structure like ModelPublicConsultant
type ModelPublicConsultant struct {
	Consultant    string `json:"consultant" bun:"consultant,type:citext,pk"`
	Ridconsultant int32  `json:"rid_consultant" bun:"rid_consultant,type:integer,pk"`
	Updatecnt     int64  `json:"updatecnt" bun:"updatecnt,type:integer,default:0"`

	MockDBAdhocBuffer `json:",omitempty"` // Embedded - RowNumber is here!
}

func TestSetRowNumbersOnRecords_RealModelStructure(t *testing.T) {
	handler := &Handler{}

	// Test with exact structure from user's ModelPublicConsultant
	records := []*ModelPublicConsultant{
		{Consultant: "John Doe", Ridconsultant: 1, Updatecnt: 0},
		{Consultant: "Jane Smith", Ridconsultant: 2, Updatecnt: 0},
		{Consultant: "Bob Johnson", Ridconsultant: 3, Updatecnt: 0},
	}

	handler.setRowNumbersOnRecords(records, 100)

	// Verify row numbers were set correctly in the embedded DBAdhocBuffer
	assert.Equal(t, int64(101), records[0].RowNumber, "First consultant should have RowNumber=101")
	assert.Equal(t, int64(102), records[1].RowNumber, "Second consultant should have RowNumber=102")
	assert.Equal(t, int64(103), records[2].RowNumber, "Third consultant should have RowNumber=103")

	t.Logf("✓ RowNumber correctly set in embedded MockDBAdhocBuffer")
	t.Logf("  Record 0: Consultant=%s, RowNumber=%d", records[0].Consultant, records[0].RowNumber)
	t.Logf("  Record 1: Consultant=%s, RowNumber=%d", records[1].Consultant, records[1].RowNumber)
	t.Logf("  Record 2: Consultant=%s, RowNumber=%d", records[2].Consultant, records[2].RowNumber)
}
