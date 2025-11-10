# Security Provider - Quick Reference

## 3-Step Setup

```go
// Step 1: Implement callbacks
func myAuth(r *http.Request) (int, string, error) { /* ... */ }
func myColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) { /* ... */ }
func myRowSec(userID int, schema, table string) (security.RowSecurity, error) { /* ... */ }

// Step 2: Configure callbacks
security.GlobalSecurity.AuthenticateCallback = myAuth
security.GlobalSecurity.LoadColumnSecurityCallback = myColSec
security.GlobalSecurity.LoadRowSecurityCallback = myRowSec

// Step 3: Setup and apply middleware
security.SetupSecurityProvider(handler, &security.GlobalSecurity)
router.Use(mux.MiddlewareFunc(security.AuthMiddleware))
router.Use(mux.MiddlewareFunc(security.SetSecurityMiddleware))
```

---

## Callback Signatures

```go
// 1. Authentication
func(r *http.Request) (userID int, roles string, err error)

// 2. Column Security
func(userID int, schema, tablename string) ([]ColumnSecurity, error)

// 3. Row Security
func(userID int, schema, tablename string) (RowSecurity, error)
```

---

## ColumnSecurity Structure

```go
security.ColumnSecurity{
    Path:       []string{"column_name"},  // ["ssn"] or ["address", "street"]
    Accesstype: "mask",                   // "mask" or "hide"
    MaskStart:  5,                        // Mask first N chars
    MaskEnd:    0,                        // Mask last N chars
    MaskChar:   "*",                      // Masking character
    MaskInvert: false,                    // true = mask middle
}
```

### Common Examples

```go
// Hide entire field
{Path: []string{"salary"}, Accesstype: "hide"}

// Mask SSN (show last 4)
{Path: []string{"ssn"}, Accesstype: "mask", MaskStart: 5}

// Mask credit card (show last 4)
{Path: []string{"credit_card"}, Accesstype: "mask", MaskStart: 12}

// Mask email (j***@example.com)
{Path: []string{"email"}, Accesstype: "mask", MaskStart: 1, MaskEnd: 0}
```

---

## RowSecurity Structure

```go
security.RowSecurity{
    Schema:    "public",
    Tablename: "orders",
    UserID:    123,
    Template:  "user_id = {UserID}",  // WHERE clause
    HasBlock:  false,                 // true = block all access
}
```

### Template Variables

- `{UserID}` - Current user ID
- `{PrimaryKeyName}` - Primary key column
- `{TableName}` - Table name
- `{SchemaName}` - Schema name

### Common Examples

```go
// Users see only their records
Template: "user_id = {UserID}"

// Users see their records OR public ones
Template: "user_id = {UserID} OR is_public = true"

// Tenant isolation
Template: "tenant_id = 5 AND user_id = {UserID}"

// Complex with subquery
Template: "dept_id IN (SELECT dept_id FROM user_depts WHERE user_id = {UserID})"

// Block all access
HasBlock: true
```

---

## Example Implementations

### Simple Header Auth

```go
func authFromHeader(r *http.Request) (int, string, error) {
    userIDStr := r.Header.Get("X-User-ID")
    if userIDStr == "" {
        return 0, "", fmt.Errorf("X-User-ID required")
    }
    userID, err := strconv.Atoi(userIDStr)
    return userID, "", err
}
```

### JWT Auth

```go
func authFromJWT(r *http.Request) (int, string, error) {
    token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
    claims, err := jwt.Parse(token, secret)
    if err != nil {
        return 0, "", err
    }
    return claims.UserID, claims.Roles, nil
}
```

### Static Column Security

```go
func loadColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    if table == "employees" {
        return []security.ColumnSecurity{
            {Path: []string{"ssn"}, Accesstype: "mask", MaskStart: 5},
            {Path: []string{"salary"}, Accesstype: "hide"},
        }, nil
    }
    return []security.ColumnSecurity{}, nil
}
```

### Database Column Security

```go
func loadColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    rows, err := db.Query(`
        SELECT control, accesstype, jsonvalue
        FROM core.secacces
        WHERE rid_hub IN (...)
        AND control ILIKE ?
    `, fmt.Sprintf("%s.%s%%", schema, table))
    // ... parse and return
}
```

### Static Row Security

```go
func loadRowSec(userID int, schema, table string) (security.RowSecurity, error) {
    templates := map[string]string{
        "orders":    "user_id = {UserID}",
        "documents": "user_id = {UserID} OR is_public = true",
    }
    return security.RowSecurity{
        Template: templates[table],
    }, nil
}
```

---

## Testing

```go
// Test auth callback
req := httptest.NewRequest("GET", "/", nil)
req.Header.Set("X-User-ID", "123")
userID, roles, err := myAuth(req)
assert.Equal(t, 123, userID)

// Test column security callback
rules, err := myColSec(123, "public", "employees")
assert.Equal(t, "mask", rules[0].Accesstype)

// Test row security callback
rowSec, err := myRowSec(123, "public", "orders")
assert.Equal(t, "user_id = {UserID}", rowSec.Template)
```

---

## Request Flow

```
HTTP Request
    ↓
AuthMiddleware → calls AuthenticateCallback
    ↓ (adds userID to context)
SetSecurityMiddleware → adds GlobalSecurity to context
    ↓
Handler.Handle()
    ↓
BeforeRead Hook → calls LoadColumnSecurityCallback + LoadRowSecurityCallback
    ↓
BeforeScan Hook → applies row security (WHERE clause)
    ↓
Database Query
    ↓
AfterRead Hook → applies column security (masking)
    ↓
HTTP Response
```

---

## Common Patterns

### Role-Based Security

```go
func loadColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    if isAdmin(userID) {
        return []security.ColumnSecurity{}, nil // No restrictions
    }
    return loadRestrictions(userID, schema, table), nil
}
```

### Tenant Isolation

```go
func loadRowSec(userID int, schema, table string) (security.RowSecurity, error) {
    tenantID := getUserTenant(userID)
    return security.RowSecurity{
        Template: fmt.Sprintf("tenant_id = %d", tenantID),
    }, nil
}
```

### Caching

```go
var cache = make(map[string][]security.ColumnSecurity)

func loadColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    key := fmt.Sprintf("%d:%s.%s", userID, schema, table)
    if cached, ok := cache[key]; ok {
        return cached, nil
    }
    rules := loadFromDB(userID, schema, table)
    cache[key] = rules
    return rules, nil
}
```

---

## Error Handling

```go
// Setup will fail if callbacks not configured
if err := security.SetupSecurityProvider(handler, &security.GlobalSecurity); err != nil {
    log.Fatal("Security setup failed:", err)
}

// Auth middleware rejects if callback returns error
func myAuth(r *http.Request) (int, string, error) {
    if invalid {
        return 0, "", fmt.Errorf("invalid credentials") // Returns HTTP 401
    }
    return userID, roles, nil
}

// Security loading can fail gracefully
func loadColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    rules, err := db.Load(...)
    if err != nil {
        log.Printf("Failed to load security: %v", err)
        return []security.ColumnSecurity{}, nil // No rules = no restrictions
    }
    return rules, nil
}
```

---

## Debugging

```go
// Enable debug logging
import "github.com/bitechdev/GoCore/pkg/cfg"
cfg.SetLogLevel("DEBUG")

// Log in callbacks
func myAuth(r *http.Request) (int, string, error) {
    token := r.Header.Get("Authorization")
    log.Printf("Auth: token=%s", token)
    // ...
}

// Check if callbacks are called
func loadColSec(userID int, schema, table string) ([]security.ColumnSecurity, error) {
    log.Printf("Loading column security: user=%d, schema=%s, table=%s", userID, schema, table)
    // ...
}
```

---

## Complete Minimal Example

```go
package main

import (
    "fmt"
    "net/http"
    "strconv"
    "github.com/bitechdev/ResolveSpec/pkg/restheadspec"
    "github.com/bitechdev/ResolveSpec/pkg/security"
    "github.com/gorilla/mux"
)

func main() {
    handler := restheadspec.NewHandlerWithGORM(db)

    // Configure callbacks
    security.GlobalSecurity.AuthenticateCallback = func(r *http.Request) (int, string, error) {
        id, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
        return id, "", nil
    }
    security.GlobalSecurity.LoadColumnSecurityCallback = func(u int, s, t string) ([]security.ColumnSecurity, error) {
        return []security.ColumnSecurity{}, nil
    }
    security.GlobalSecurity.LoadRowSecurityCallback = func(u int, s, t string) (security.RowSecurity, error) {
        return security.RowSecurity{Template: fmt.Sprintf("user_id = %d", u)}, nil
    }

    // Setup
    security.SetupSecurityProvider(handler, &security.GlobalSecurity)

    // Middleware
    router := mux.NewRouter()
    restheadspec.SetupMuxRoutes(router, handler)
    router.Use(mux.MiddlewareFunc(security.AuthMiddleware))
    router.Use(mux.MiddlewareFunc(security.SetSecurityMiddleware))

    http.ListenAndServe(":8080", router)
}
```

---

## Resources

| File | Description |
|------|-------------|
| `CALLBACKS_GUIDE.md` | **Start here** - Complete implementation guide |
| `callbacks_example.go` | 7 working examples to copy |
| `CALLBACKS_SUMMARY.md` | Architecture overview |
| `README.md` | Full documentation |
| `setup_example.go` | Integration examples |

---

## Cheat Sheet

```go
// ===== REQUIRED SETUP =====
security.GlobalSecurity.AuthenticateCallback = myAuthFunc
security.GlobalSecurity.LoadColumnSecurityCallback = myColFunc
security.GlobalSecurity.LoadRowSecurityCallback = myRowFunc
security.SetupSecurityProvider(handler, &security.GlobalSecurity)

// ===== CALLBACK SIGNATURES =====
func(r *http.Request) (int, string, error)                         // Auth
func(int, string, string) ([]security.ColumnSecurity, error)       // Column
func(int, string, string) (security.RowSecurity, error)            // Row

// ===== QUICK EXAMPLES =====
// Header auth
func(r *http.Request) (int, string, error) {
    id, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
    return id, "", nil
}

// Mask SSN
{Path: []string{"ssn"}, Accesstype: "mask", MaskStart: 5}

// User isolation
{Template: "user_id = {UserID}"}
```
