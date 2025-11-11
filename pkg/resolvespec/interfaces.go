package resolvespec

// Legacy interfaces for backward compatibility
type GormTableNameInterface interface {
	TableName() string
}

type GormTableSchemaInterface interface {
	TableSchema() string
}

type GormTableCRUDRequest struct {
	CRUDRequest *string `json:"crud_request"`
}

func (r *GormTableCRUDRequest) SetRequest(request string) {
	r.CRUDRequest = &request
}

func (r GormTableCRUDRequest) GetRequest() string {
	return *r.CRUDRequest
}

// New interfaces that replace the legacy ones above
// These are now defined in database.go:
// - TableNameProvider (replaces GormTableNameInterface)
// - SchemaProvider (replaces GormTableSchemaInterface)
