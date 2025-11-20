# X-Files Header Usage

The `x-files` header allows you to configure complex query options using a single JSON object. The XFiles configuration is parsed and populates the `ExtendedRequestOptions` fields, which means it integrates seamlessly with the existing query building system.

## Architecture

When an `x-files` header is received:
1. It's parsed into an `XFiles` struct
2. The `XFiles` fields populate the `ExtendedRequestOptions` (columns, filters, sort, preload, etc.)
3. The normal query building process applies these options to the SQL query
4. This allows x-files to work alongside individual headers if needed

## Basic Example

```http
GET /public/users
X-Files: {"tablename":"users","columns":["id","name","email"],"limit":"10","offset":"0"}
```

## Complete Example

```http
GET /public/users
X-Files: {
  "tablename": "users",
  "schema": "public",
  "columns": ["id", "name", "email", "created_at"],
  "omit_columns": [],
  "sort": ["-created_at", "name"],
  "limit": "50",
  "offset": "0",
  "filter_fields": [
    {
      "field": "status",
      "operator": "eq",
      "value": "active"
    },
    {
      "field": "age",
      "operator": "gt",
      "value": "18"
    }
  ],
  "sql_and": ["deleted_at IS NULL"],
  "sql_or": [],
  "cql_columns": ["UPPER(name)"],
  "skipcount": false,
  "distinct": false
}
```

## Supported Filter Operators

- `eq` - equals
- `neq` - not equals
- `gt` - greater than
- `gte` - greater than or equals
- `lt` - less than
- `lte` - less than or equals
- `like` - SQL LIKE
- `ilike` - case-insensitive LIKE
- `in` - IN clause
- `between` - between (exclusive)
- `between_inclusive` - between (inclusive)
- `is_null` - is NULL
- `is_not_null` - is NOT NULL

## Sorting

Sort fields can be prefixed with:
- `+` for ascending (default)
- `-` for descending

Examples:
- `"sort": ["name"]` - ascending by name
- `"sort": ["-created_at"]` - descending by created_at
- `"sort": ["-created_at", "name"]` - multiple sorts

## Computed Columns (CQL)

Use `cql_columns` to add computed SQL expressions:

```json
{
  "cql_columns": [
    "UPPER(name)",
    "CONCAT(first_name, ' ', last_name)"
  ]
}
```

These will be available as `cql1`, `cql2`, etc. in the response.

## Cursor Pagination

```json
{
  "cursor_forward": "eyJpZCI6MTAwfQ==",
  "cursor_backward": ""
}
```

## Base64 Encoding

For complex JSON, you can base64-encode the value and prefix it with `ZIP_` or `__`:

```http
GET /public/users
X-Files: ZIP_eyJ0YWJsZW5hbWUiOiJ1c2VycyIsImxpbWl0IjoiMTAifQ==
```

## XFiles Struct Reference

```go
type XFiles struct {
    TableName      string      `json:"tablename"`
    Schema         string      `json:"schema"`
    PrimaryKey     string      `json:"primarykey"`
    ForeignKey     string      `json:"foreignkey"`
    RelatedKey     string      `json:"relatedkey"`
    Sort           []string    `json:"sort"`
    Prefix         string      `json:"prefix"`
    Editable       bool        `json:"editable"`
    Recursive      bool        `json:"recursive"`
    Expand         bool        `json:"expand"`
    Rownumber      bool        `json:"rownumber"`
    Skipcount      bool        `json:"skipcount"`
    Offset         json.Number `json:"offset"`
    Limit          json.Number `json:"limit"`
    Columns        []string    `json:"columns"`
    OmitColumns    []string    `json:"omit_columns"`
    CQLColumns     []string    `json:"cql_columns"`
    SqlJoins       []string    `json:"sql_joins"`
    SqlOr          []string    `json:"sql_or"`
    SqlAnd         []string    `json:"sql_and"`
    FilterFields   []struct {
        Field    string `json:"field"`
        Value    string `json:"value"`
        Operator string `json:"operator"`
    } `json:"filter_fields"`
    CursorForward  string `json:"cursor_forward"`
    CursorBackward string `json:"cursor_backward"`
}
```

## Recursive Preloading with ParentTables and ChildTables

XFiles now supports recursive preloading of related entities:

```json
{
  "tablename": "users",
  "columns": ["id", "name"],
  "limit": "10",
  "parenttables": [
    {
      "tablename": "Company",
      "columns": ["id", "name", "industry"],
      "sort": ["-created_at"]
    }
  ],
  "childtables": [
    {
      "tablename": "Orders",
      "columns": ["id", "total", "status"],
      "limit": "5",
      "sort": ["-order_date"],
      "filter_fields": [
        {"field": "status", "operator": "eq", "value": "completed"}
      ],
      "childtables": [
        {
          "tablename": "OrderItems",
          "columns": ["id", "product_name", "quantity"],
          "recursive": true
        }
      ]
    }
  ]
}
```

### How Recursive Preloading Works

- **ParentTables**: Preloads parent relationships (e.g., User -> Company)
- **ChildTables**: Preloads child relationships (e.g., User -> Orders -> OrderItems)
- **Recursive**: When `true`, continues preloading the same relation recursively
- Each nested table can have its own:
  - Column selection (`columns`, `omit_columns`)
  - Filtering (`filter_fields`, `sql_and`)
  - Sorting (`sort`)
  - Pagination (`limit`)
  - Further nesting (`parenttables`, `childtables`)

### Relation Path Building

Relations are built as dot-separated paths:
- `Company` (direct parent)
- `Orders` (direct child)
- `Orders.OrderItems` (nested child)
- `Orders.OrderItems.Product` (deeply nested)

## Notes

- Individual headers (like `x-select-fields`, `x-sort`, etc.) can still be used alongside `x-files`
- X-Files populates `ExtendedRequestOptions` which is then processed by the normal query building logic
- ParentTables and ChildTables are converted to `PreloadOption` entries with full support for:
  - Column selection
  - Filtering
  - Sorting
  - Limit
  - Recursive nesting
- The relation name in ParentTables/ChildTables should match the GORM/Bun relation field name on the model
