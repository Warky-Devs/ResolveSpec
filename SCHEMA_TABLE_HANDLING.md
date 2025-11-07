# Schema and Table Name Handling

This document explains how the handlers properly separate and handle schema and table names.

## Implementation

Both `resolvespec` and `restheadspec` handlers now properly handle schema and table name separation through the following functions:

- `parseTableName(fullTableName)` - Splits "schema.table" into separate components
- `getSchemaAndTable(defaultSchema, entity, model)` - Returns schema and table separately
- `getTableName(schema, entity, model)` - Returns the full "schema.table" format

## Priority Order

When determining the schema and table name, the following priority is used:

1. **If `TableName()` contains a schema** (e.g., "myschema.mytable"), that schema takes precedence
2. **If model implements `SchemaProvider`**, use that schema
3. **Otherwise**, use the `defaultSchema` parameter from the URL/request

## Scenarios

### Scenario 1: Simple table name, default schema
```go
type User struct {
    ID   string
    Name string
}

func (User) TableName() string {
    return "users"
}
```
- Request URL: `/api/public/users`
- Result: `schema="public"`, `table="users"`, `fullName="public.users"`

### Scenario 2: Table name includes schema
```go
type User struct {
    ID   string
    Name string
}

func (User) TableName() string {
    return "auth.users"  // Schema included!
}
```
- Request URL: `/api/public/users` (public is ignored)
- Result: `schema="auth"`, `table="users"`, `fullName="auth.users"`
- **Note**: The schema from `TableName()` takes precedence over the URL schema

### Scenario 3: Using SchemaProvider
```go
type User struct {
    ID   string
    Name string
}

func (User) TableName() string {
    return "users"
}

func (User) SchemaName() string {
    return "auth"
}
```
- Request URL: `/api/public/users` (public is ignored)
- Result: `schema="auth"`, `table="users"`, `fullName="auth.users"`

### Scenario 4: Table name includes schema AND SchemaProvider
```go
type User struct {
    ID   string
    Name string
}

func (User) TableName() string {
    return "core.users"  // This wins!
}

func (User) SchemaName() string {
    return "auth"  // This is ignored
}
```
- Request URL: `/api/public/users`
- Result: `schema="core"`, `table="users"`, `fullName="core.users"`
- **Note**: Schema from `TableName()` takes highest precedence

### Scenario 5: No providers at all
```go
type User struct {
    ID   string
    Name string
}
// No TableName() or SchemaName()
```
- Request URL: `/api/public/users`
- Result: `schema="public"`, `table="users"`, `fullName="public.users"`
- Uses URL schema and entity name

## Key Features

1. **Automatic detection**: The code automatically detects if `TableName()` includes a schema by checking for "."
2. **Backward compatible**: Existing code continues to work
3. **Flexible**: Supports multiple ways to specify schema and table
4. **Debug logging**: Logs when schema is detected in `TableName()` for debugging

## Code Locations

### Handlers
- `/pkg/resolvespec/handler.go:472-531`
- `/pkg/restheadspec/handler.go:534-593`

### Database Adapters
- `/pkg/common/adapters/database/utils.go` - Shared `parseTableName()` function
- `/pkg/common/adapters/database/bun.go` - Bun adapter with separated schema/table
- `/pkg/common/adapters/database/gorm.go` - GORM adapter with separated schema/table

## Adapter Implementation

Both Bun and GORM adapters now properly separate schema and table name:

```go
// BunSelectQuery/GormSelectQuery now have separated fields:
type BunSelectQuery struct {
    query      *bun.SelectQuery
    schema     string // Separated schema name
    tableName  string // Just the table name, without schema
    tableAlias string
}
```

When `Model()` or `Table()` is called:
1. The full table name (which may include schema) is parsed
2. Schema and table name are stored separately
3. When building joins, the already-separated table name is used directly

This ensures consistent handling of schema-qualified table names throughout the codebase.
