package resolvespec

import "context"

// Database interface designed to work with both GORM and Bun
type Database interface {
	// Core query operations
	NewSelect() SelectQuery
	NewInsert() InsertQuery
	NewUpdate() UpdateQuery
	NewDelete() DeleteQuery
	
	// Raw SQL execution
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	
	// Transaction support
	BeginTx(ctx context.Context) (Database, error)
	CommitTx(ctx context.Context) error
	RollbackTx(ctx context.Context) error
	RunInTransaction(ctx context.Context, fn func(Database) error) error
}

// SelectQuery interface for building SELECT queries (compatible with both GORM and Bun)
type SelectQuery interface {
	Model(model interface{}) SelectQuery
	Table(table string) SelectQuery
	Column(columns ...string) SelectQuery
	Where(query string, args ...interface{}) SelectQuery
	WhereOr(query string, args ...interface{}) SelectQuery
	Join(query string, args ...interface{}) SelectQuery
	LeftJoin(query string, args ...interface{}) SelectQuery
	Order(order string) SelectQuery
	Limit(n int) SelectQuery
	Offset(n int) SelectQuery
	Group(group string) SelectQuery
	Having(having string, args ...interface{}) SelectQuery
	
	// Execution methods
	Scan(ctx context.Context, dest interface{}) error
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context) (bool, error)
}

// InsertQuery interface for building INSERT queries
type InsertQuery interface {
	Model(model interface{}) InsertQuery
	Table(table string) InsertQuery
	Value(column string, value interface{}) InsertQuery
	OnConflict(action string) InsertQuery
	Returning(columns ...string) InsertQuery
	
	// Execution
	Exec(ctx context.Context) (Result, error)
}

// UpdateQuery interface for building UPDATE queries
type UpdateQuery interface {
	Model(model interface{}) UpdateQuery
	Table(table string) UpdateQuery
	Set(column string, value interface{}) UpdateQuery
	SetMap(values map[string]interface{}) UpdateQuery
	Where(query string, args ...interface{}) UpdateQuery
	Returning(columns ...string) UpdateQuery
	
	// Execution
	Exec(ctx context.Context) (Result, error)
}

// DeleteQuery interface for building DELETE queries
type DeleteQuery interface {
	Model(model interface{}) DeleteQuery
	Table(table string) DeleteQuery
	Where(query string, args ...interface{}) DeleteQuery
	
	// Execution
	Exec(ctx context.Context) (Result, error)
}

// Result interface for query execution results
type Result interface {
	RowsAffected() int64
	LastInsertId() (int64, error)
}

// ModelRegistry manages model registration and retrieval
type ModelRegistry interface {
	RegisterModel(name string, model interface{}) error
	GetModel(name string) (interface{}, error)
	GetAllModels() map[string]interface{}
	GetModelByEntity(schema, entity string) (interface{}, error)
}

// Router interface for HTTP router abstraction
type Router interface {
	HandleFunc(pattern string, handler HTTPHandlerFunc) RouteRegistration
	ServeHTTP(w ResponseWriter, r *Request)
}

// RouteRegistration allows method chaining for route configuration
type RouteRegistration interface {
	Methods(methods ...string) RouteRegistration
	PathPrefix(prefix string) RouteRegistration
}

// Request interface abstracts HTTP request
type Request interface {
	Method() string
	URL() string
	Header(key string) string
	Body() ([]byte, error)
	PathParam(key string) string
	QueryParam(key string) string
}

// ResponseWriter interface abstracts HTTP response
type ResponseWriter interface {
	SetHeader(key, value string)
	WriteHeader(statusCode int)
	Write(data []byte) (int, error)
	WriteJSON(data interface{}) error
}

// HTTPHandlerFunc type for HTTP handlers  
type HTTPHandlerFunc func(ResponseWriter, Request)

// TableNameProvider interface for models that provide table names
type TableNameProvider interface {
	TableName() string
}

// SchemaProvider interface for models that provide schema names  
type SchemaProvider interface {
	SchemaName() string
}