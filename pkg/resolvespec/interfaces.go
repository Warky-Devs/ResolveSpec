package resolvespec

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
