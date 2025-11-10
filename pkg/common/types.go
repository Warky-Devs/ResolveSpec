package common

type RequestBody struct {
	Operation string         `json:"operation"`
	Data      interface{}    `json:"data"`
	ID        *int64         `json:"id"`
	Options   RequestOptions `json:"options"`
}

type RequestOptions struct {
	Preload         []PreloadOption  `json:"preload"`
	Columns         []string         `json:"columns"`
	OmitColumns     []string         `json:"omit_columns"`
	Filters         []FilterOption   `json:"filters"`
	Sort            []SortOption     `json:"sort"`
	Limit           *int             `json:"limit"`
	Offset          *int             `json:"offset"`
	CustomOperators []CustomOperator `json:"customOperators"`
	ComputedColumns []ComputedColumn `json:"computedColumns"`
	Parameters      []Parameter      `json:"parameters"`

	// Cursor pagination
	CursorForward  string  `json:"cursor_forward"`
	CursorBackward string  `json:"cursor_backward"`
	FetchRowNumber *string `json:"fetch_row_number"`
}

type Parameter struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Sequence *int   `json:"sequence"`
}

type PreloadOption struct {
	Relation    string         `json:"relation"`
	Columns     []string       `json:"columns"`
	OmitColumns []string       `json:"omit_columns"`
	Filters     []FilterOption `json:"filters"`
	Limit       *int           `json:"limit"`
	Offset      *int           `json:"offset"`
	Updatable   *bool          `json:"updateable"` // if true, the relation can be updated
}

type FilterOption struct {
	Column        string      `json:"column"`
	Operator      string      `json:"operator"`
	Value         interface{} `json:"value"`
	LogicOperator string      `json:"logic_operator"` // "AND" or "OR" - how this filter combines with previous filters
}

type SortOption struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

type CustomOperator struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

type ComputedColumn struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// Response structures
type Response struct {
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Metadata *Metadata   `json:"metadata,omitempty"`
	Error    *APIError   `json:"error,omitempty"`
}

type Metadata struct {
	Total     int64  `json:"total"`
	Count     int64  `json:"count"`
	Filtered  int64  `json:"filtered"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	RowNumber *int64 `json:"row_number,omitempty"`
}

type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Detail  string      `json:"detail,omitempty"`
}

type Column struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsNullable bool   `json:"is_nullable"`
	IsPrimary  bool   `json:"is_primary"`
	IsUnique   bool   `json:"is_unique"`
	HasIndex   bool   `json:"has_index"`
}

type TableMetadata struct {
	Schema    string   `json:"schema"`
	Table     string   `json:"table"`
	Columns   []Column `json:"columns"`
	Relations []string `json:"relations"`
}
