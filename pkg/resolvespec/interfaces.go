package resolvespec

type GormTableNameInterface interface {
	TableName() string
}

type GormTableSchemaInterface interface {
	TableSchema() string
}
