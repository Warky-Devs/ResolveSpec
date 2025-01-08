# 📜 ResolveSpec 📜

ResolveSpec is a flexible and powerful REST API specification and implementation that provides GraphQL-like capabilities while maintaining REST simplicity. It allows for dynamic data querying, relationship preloading, and complex filtering through a clean, URL-based interface.

![slogan](./generated_slogan.webp)

## Features

- **Dynamic Data Querying**: Select specific columns and relationships to return
- **Relationship Preloading**: Load related entities with custom column selection and filters
- **Complex Filtering**: Apply multiple filters with various operators
- **Sorting**: Multi-column sort support
- **Pagination**: Built-in limit and offset support
- **Computed Columns**: Define virtual columns for complex calculations
- **Custom Operators**: Add custom SQL conditions when needed

## API Structure

### URL Patterns
```
/[schema]/[table_or_entity]/[id]
/[schema]/[table_or_entity]
/[schema]/[function]
/[schema]/[virtual]
```

### Request Format

```json
{
  "operation": "read|create|update|delete",
  "data": {
    // For create/update operations
  },
  "options": {
    "preload": [...],
    "columns": [...],
    "filters": [...],
    "sort": [...],
    "limit": number,
    "offset": number,
    "customOperators": [...],
    "computedColumns": [...]
  }
}
```

## Example Usage

### Reading Data with Related Entities
```json
POST /core/users
{
  "operation": "read",
  "options": {
    "columns": ["id", "name", "email"],
    "preload": [
      {
        "relation": "posts",
        "columns": ["id", "title"],
        "filters": [
          {
            "column": "status",
            "operator": "eq",
            "value": "published"
          }
        ]
      }
    ],
    "filters": [
      {
        "column": "active",
        "operator": "eq",
        "value": true
      }
    ],
    "sort": [
      {
        "column": "created_at",
        "direction": "desc"
      }
    ],
    "limit": 10,
    "offset": 0
  }
}
```

## Installation

```bash
go get github.com/Warky-Devs/ResolveSpec
```

## Quick Start

1. Import the package:
```go
import "github.com/Warky-Devs/ResolveSpec"
```

1. Initialize the handler:
```go
handler := resolvespec.NewAPIHandler(db)

// Register your models
handler.RegisterModel("core", "users", &User{})
handler.RegisterModel("core", "posts", &Post{})
```

3. Use with your preferred router:

Using Gin:
```go
func setupGin(handler *resolvespec.APIHandler) *gin.Engine {
    r := gin.Default()
    
    r.POST("/:schema/:entity", func(c *gin.Context) {
        params := map[string]string{
            "schema": c.Param("schema"),
            "entity": c.Param("entity"),
            "id":     c.Param("id"),
        }
        handler.SetParams(params)
        handler.Handle(c.Writer, c.Request)
    })
    
    return r
}
```

Using Mux:
```go
func setupMux(handler *resolvespec.APIHandler) *mux.Router {
    r := mux.NewRouter()
    
    r.HandleFunc("/{schema}/{entity}", func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        handler.SetParams(vars)
        handler.Handle(w, r)
    }).Methods("POST")
    
    return r
}
```

## Configuration

### Model Registration
```go
type User struct {
    ID    uint   `json:"id" gorm:"primaryKey"`
    Name  string `json:"name"`
    Email string `json:"email"`
    Posts []Post `json:"posts,omitempty" gorm:"foreignKey:UserID"`
}

handler.RegisterModel("core", "users", &User{})
```

## Features in Detail

### Filtering
Supported operators:
- eq: Equal
- neq: Not Equal
- gt: Greater Than
- gte: Greater Than or Equal
- lt: Less Than
- lte: Less Than or Equal
- like: LIKE pattern matching
- ilike: Case-insensitive LIKE
- in: IN clause

### Sorting
Support for multiple sort criteria with direction:
```json
"sort": [
  {
    "column": "created_at",
    "direction": "desc"
  },
  {
    "column": "name",
    "direction": "asc"
  }
]
```

### Computed Columns
Define virtual columns using SQL expressions:
```json
"computedColumns": [
  {
    "name": "full_name",
    "expression": "CONCAT(first_name, ' ', last_name)"
  }
]
```

## Security Considerations

- Implement proper authentication and authorization
- Validate all input parameters
- Use prepared statements (handled by GORM)
- Implement rate limiting
- Control access at schema/entity level

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 

## Acknowledgments

- Inspired by REST, Odata and GraphQL's flexibility
- Built with [GORM](https://gorm.io)
- Uses Gin or Mux Web Framework
- Slogan generated using DALL-E
- AI used for documentation checking and correction